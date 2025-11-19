package provider

import (
	"context"
	"crypto/sha1"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/99designs/keyring"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api"
	"github.com/synology-community/terraform-provider-synology/synology/provider/container"
	"github.com/synology-community/terraform-provider-synology/synology/provider/core"
	"github.com/synology-community/terraform-provider-synology/synology/provider/filestation"
	"github.com/synology-community/terraform-provider-synology/synology/provider/virtualization"
	"golang.org/x/sync/singleflight"
)

const (
	SYNOLOGY_HOST_ENV_VAR            = "SYNOLOGY_HOST"
	SYNOLOGY_USER_ENV_VAR            = "SYNOLOGY_USER"
	SYNOLOGY_PASSWORD_ENV_VAR        = "SYNOLOGY_PASSWORD"
	SYNOLOGY_OTP_SECRET_ENV_VAR      = "SYNOLOGY_OTP_SECRET"
	SYNOLOGY_SKIP_CERT_CHECK_ENV_VAR = "SYNOLOGY_SKIP_CERT_CHECK"
	SYNOLOGY_SESSION_CACHE_MODE      = "SYNOLOGY_SESSION_CACHE"      // auto | keyring | file | memory | off
	SYNOLOGY_SESSION_CACHE_PATH      = "SYNOLOGY_SESSION_CACHE_PATH" // when mode=file
)

// singleflight deduplicates concurrent logins within a single provider process.
var (
	sf singleflight.Group
)

// cache key for a provider instance.
func sessionKey(host, user string, skipCertCheck bool) string {
	return host + "\x00" + user + "\x00" + strconv.FormatBool(skipCertCheck)
}

// sessionRecord keeps info from c.ExportSession plus last TOTP step to avoid replay.
type sessionRecord struct {
	SessionID    string    `json:"sid"`
	SynoToken    string    `json:"syno_token"`
	IssuedAt     time.Time `json:"issued_at"`
	LastTotpStep int64     `json:"last_totp_step"`
}

// cacheKey returns a stable key for storing session info for a given provider instance.
func cacheKey(host, user string, skipCertCheck bool) string {
	sum := sha1.Sum([]byte(sessionKey(host, user, skipCertCheck)))
	return "synology:" + hex.EncodeToString(sum[:])
}

// openSessionRing tries to open a keyring according to the provided mode and path.
func openSessionRing(mode, path string) (keyring.Keyring, string, error) {
	cfg := keyring.Config{ServiceName: "terraform-provider-synology"}
	switch mode {
	case "auto":
		cfg.AllowedBackends = []keyring.BackendType{
			keyring.KeychainBackend, keyring.WinCredBackend,
			keyring.SecretServiceBackend, keyring.KWalletBackend,
			keyring.PassBackend, keyring.FileBackend,
		}
	case "keyring":
		cfg.AllowedBackends = []keyring.BackendType{
			keyring.KeychainBackend, keyring.WinCredBackend,
			keyring.SecretServiceBackend, keyring.KWalletBackend,
		}
	case "file":
		cfg.AllowedBackends = []keyring.BackendType{keyring.FileBackend}
	case "memory":
		// handled later
	default: // off
		return nil, "", fmt.Errorf("disabled")
	}
	if path == "" {
		if dir, err := os.UserCacheDir(); err == nil {
			path = filepath.Join(dir, "terraform-provider-synology", "sessions")
		}
	}
	if path != "" {
		_ = os.MkdirAll(path, 0o700)
		cfg.FileDir = path
	}
	if len(cfg.AllowedBackends) > 0 {
		if r, err := keyring.Open(cfg); err == nil {
			return r, cfg.FileDir, nil
		}
	}
	// fallback to in-memory so runs still work even without persistence.
	return keyring.NewArrayKeyring(nil), "", nil
}

// readSession reads a sessionRecord from the keyring.
func readSession(r keyring.Keyring, key string) (*sessionRecord, error) {
	if r == nil {
		return nil, fmt.Errorf("no keyring")
	}
	it, err := r.Get(key)
	if err != nil {
		return nil, err
	}
	var rec sessionRecord
	if err := json.Unmarshal(it.Data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// writeSession writes a sessionRecord to the keyring.
func writeSession(r keyring.Keyring, key string, rec sessionRecord) error {
	if r == nil {
		return fmt.Errorf("no keyring")
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return r.Set(keyring.Item{
		Key:                         key,
		Data:                        b,
		Label:                       "Synology session for Terraform provider",
		KeychainNotTrustApplication: true,
	})
}

// nextTotpWaitDuration returns the time until the next 30s TOTP boundary (+ small guard).
func nextTotpWaitDuration() time.Duration {
	now := time.Now()
	sec := now.Unix() % 30
	wait := time.Duration(30-sec) * time.Second
	return wait + 150*time.Millisecond
}

// waitUntilNextTotpStep sleeps for the provided duration (typically until the next 30s TOTP boundary).
func waitUntilNextTotpStep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Ensure SynologyProvider satisfies various provider interfaces.
var _ provider.Provider = &SynologyProvider{}

// SynologyProvider defines the provider implementation.
type SynologyProvider struct{}

// SessionCacheModel describes the session cache configuration.
type SessionCacheModel struct {
	Mode types.String `tfsdk:"mode"`
	Path types.String `tfsdk:"path"`
}

// SynologyProviderModel describes the provider data model.
type SynologyProviderModel struct {
	Host          types.String `tfsdk:"host"`
	User          types.String `tfsdk:"user"`
	Password      types.String `tfsdk:"password"`
	OtpSecret     types.String `tfsdk:"otp_secret"`
	SkipCertCheck types.Bool   `tfsdk:"skip_cert_check"`
	SessionCache  types.Object `tfsdk:"session_cache"`
}

var (
	reBase32Chars = regexp.MustCompile(`^[A-Z2-7= ]+$`)
	reHostname    = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
)

// getStringWithEnvFallback returns the value from a types.String attribute, falling back to an environment variable if unset.
func getStringWithEnvFallback(attr types.String, envVar string) string {
	if !attr.IsNull() && !attr.IsUnknown() && attr.ValueString() != "" {
		return attr.ValueString()
	}
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return ""
}

// getBoolWithEnvFallback returns the value from a types.Bool attribute, falling back to an environment variable if unset.
func getBoolWithEnvFallback(attr types.Bool, envVar string, defaultValue bool) bool {
	if !attr.IsNull() && !attr.IsUnknown() {
		return attr.ValueBool()
	}
	if vString := os.Getenv(envVar); vString != "" {
		if v, err := strconv.ParseBool(vString); err == nil {
			return v
		}
	}
	return defaultValue
}

func (p *SynologyProvider) Metadata(
	ctx context.Context,
	req provider.MetadataRequest,
	resp *provider.MetadataResponse,
) {
	resp.TypeName = "synology"

	tflog.Info(ctx, "Starting")
}

func (p *SynologyProvider) Schema(
	ctx context.Context,
	req provider.SchemaRequest,
	resp *provider.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "The Synology provider enables Terraform to manage resources on Synology DiskStation Manager (DSM) systems.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "Remote Synology URL, e.g. `https://host:5001`.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						reHostname,
						"must be a valid URL",
					),
				},
			},
			"user": schema.StringAttribute{
				Description: "User to connect to Synology station with.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Password to use when connecting to Synology station.",
				Optional:    true,
				Sensitive:   true,
			},
			"otp_secret": schema.StringAttribute{
				Description: "OTP secret to use when connecting to Synology station (valid RFC 4648 base32 TOTP secret: A-Z, 2-7, optional '=', spaces ignored).",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(16, 32),
					stringvalidator.RegexMatches(
						reBase32Chars,
						"must be base32 encoded secret only, not whole otp:// URI",
					),
				},
			},
			"skip_cert_check": schema.BoolAttribute{
				Description: "Whether to skip SSL certificate checks.",
				Optional:    true,
			},
			"session_cache": schema.SingleNestedAttribute{
				Description: "Session cache configuration. Supports caching Synology DSM sessions to reduce login frequency.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						Description: "Session cache mode - one of: auto, keyring, file, memory, off. Default: off. Can be set via SYNOLOGY_SESSION_CACHE environment variable.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("auto", "keyring", "file", "memory", "off"),
						},
					},
					"path": schema.StringAttribute{
						Description: "Directory for file-based session cache when mode = \"file\". Defaults to OS user cache dir. Can be set via SYNOLOGY_SESSION_CACHE_PATH environment variable.",
						Optional:    true,
					},
				},
			},
		},
	}
}

func (p *SynologyProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	var data SynologyProviderModel

	tflog.Info(ctx, "Configuring Synology provider")

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get configuration values with environment variable fallbacks
	host := getStringWithEnvFallback(data.Host, SYNOLOGY_HOST_ENV_VAR)
	user := getStringWithEnvFallback(data.User, SYNOLOGY_USER_ENV_VAR)
	password := getStringWithEnvFallback(data.Password, SYNOLOGY_PASSWORD_ENV_VAR)
	otpSecret := getStringWithEnvFallback(data.OtpSecret, SYNOLOGY_OTP_SECRET_ENV_VAR)
	skipCertificateCheck := getBoolWithEnvFallback(
		data.SkipCertCheck,
		SYNOLOGY_SKIP_CERT_CHECK_ENV_VAR,
		true,
	)

	// Validate OTP secret only if it's provided
	if otpSecret != "" {
		if err := validateOtpSecret(otpSecret); err != nil {
			resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
				path.Root("otp_secret"),
				"Invalid OTP secret",
				err.Error(),
			))
			return
		}
	}

	// Extract session cache configuration
	var sessionCache SessionCacheModel
	cacheMode := "off"
	cachePath := ""

	if !data.SessionCache.IsNull() && !data.SessionCache.IsUnknown() {
		diags := data.SessionCache.As(
			ctx,
			&sessionCache,
			basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true},
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		cacheMode = getStringWithEnvFallback(sessionCache.Mode, SYNOLOGY_SESSION_CACHE_MODE)
		cachePath = getStringWithEnvFallback(sessionCache.Path, SYNOLOGY_SESSION_CACHE_PATH)
	} else {
		// Check environment variables if session_cache object is not provided
		if v := os.Getenv(SYNOLOGY_SESSION_CACHE_MODE); v != "" {
			cacheMode = v
		}
		if v := os.Getenv(SYNOLOGY_SESSION_CACHE_PATH); v != "" {
			cachePath = v
		}
	}

	// Use default if still empty
	if cacheMode == "" {
		cacheMode = "off"
	}

	// Validate cache mode
	if !isValidSessionCacheMode(cacheMode) {
		resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
			path.Root("session_cache").AtName("mode"),
			"Invalid session cache mode",
			fmt.Sprintf(
				"Unsupported value %q; valid values are: auto, keyring, file, memory, off.",
				cacheMode,
			),
		))
		return
	}

	if host == "" {
		resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
			path.Root("host"),
			"invalid provider configuration",
			"host information is not provided"))
	}
	if user == "" {
		resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
			path.Root("user"),
			"invalid provider configuration",
			"user information is not provided"))
	}
	if password == "" {
		resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
			path.Root("password"),
			"invalid provider configuration",
			"password information is not provided"))
	}

	if resp.Diagnostics.HasError() {
		return
	}

	key := sessionKey(host, user, skipCertificateCheck)
	cacheK := cacheKey(host, user, skipCertificateCheck)

	ring, _, _ := openSessionRing(cacheMode, cachePath)

	cli, err := client.New(api.Options{
		Host:       host,
		VerifyCert: !skipCertificateCheck,
		RetryLimit: 5,
		Logger:     NewLogger(ctx),
	})
	if err != nil {
		resp.Diagnostics.AddError("failed to initialize Synology client", err.Error())
		return
	}

	if ring != nil {
		if rec, err := readSession(ring, cacheK); err == nil && rec != nil && rec.SessionID != "" &&
			rec.SynoToken != "" {
			if c, ok := cli.(*client.Client); ok {
				c.ImportSession(
					api.Session{
						SessionID: rec.SessionID,
						SynoToken: rec.SynoToken,
						CreatedAt: rec.IssuedAt,
					},
				)
				if alive, _ := c.IsSessionAlive(ctx); alive {
					tflog.Info(ctx, "reused cached Synology session")
					resp.DataSourceData = c
					resp.ResourceData = c
					return
				}
				tflog.Info(ctx, "cached Synology session expired")
			}
		}
	}

	v, err, _ := sf.Do(key, func() (any, error) {
		if otpSecret != "" {
			wait := nextTotpWaitDuration()
			tflog.Warn(ctx, fmt.Sprintf("Waiting %s for fresh TOTP window before login.", wait))
			_ = waitUntilNextTotpStep(ctx, wait)
		}

		_, loginErr := cli.Login(ctx, api.LoginOptions{
			Username:  user,
			Password:  password,
			OTPSecret: otpSecret,
		})
		if loginErr == api.ErrOtpRejected && otpSecret != "" {
			wait := nextTotpWaitDuration()
			tflog.Warn(
				ctx,
				fmt.Sprintf(
					"Synology OTP rejected — waiting %s before retrying to avoid TOTP replay.",
					wait,
				),
			)
			resp.Diagnostics.Append(diag.NewWarningDiagnostic(
				"Synology OTP rejected — wait",
				fmt.Sprintf(
					"DSM rejected the OTP (likely replay within the same 30‑second window). Started %s wait before retrying with a fresh code.",
					wait,
				),
			))
			if werr := waitUntilNextTotpStep(ctx, wait); werr == nil {
				_, loginErr = cli.Login(ctx, api.LoginOptions{
					Username:  user,
					Password:  password,
					OTPSecret: otpSecret,
				})
			}
		}
		if loginErr != nil {
			if cli.Credentials().Token == "" {
				return nil, loginErr
			}
		}
		if c, ok := cli.(*client.Client); ok && ring != nil {
			s := c.ExportSession()
			_ = writeSession(ring, cacheK, sessionRecord{
				SessionID:    s.SessionID,
				SynoToken:    s.SynoToken,
				IssuedAt:     s.CreatedAt,
				LastTotpStep: time.Now().Unix() / 30,
			})
		}
		return cli, nil
	})
	if err != nil {
		resp.Diagnostics.Append(
			diag.NewErrorDiagnostic(
				"login to Synology station failed",
				fmt.Sprintf("Unable to login to Synology station, got error: %s", err),
			),
		)
		return
	}

	if c, ok := v.(*client.Client); ok {
		resp.DataSourceData = c
		resp.ResourceData = c
		return
	}

	if cli == nil {
		resp.Diagnostics.AddError(
			"internal error",
			"unexpected error: nil client after login",
		)
		return
	}

	if c, ok := cli.(*client.Client); ok {
		resp.DataSourceData = c
		resp.ResourceData = c
	}
}

func (p *SynologyProvider) Resources(ctx context.Context) []func() resource.Resource {
	var resp []func() resource.Resource

	resp = append(resp, NewApiResource, NewPasswordResource)
	resp = append(resp, core.Resources()...)
	resp = append(resp, filestation.Resources()...)
	resp = append(resp, virtualization.Resources()...)
	resp = append(resp, container.Resources()...)

	return resp
}

func (p *SynologyProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	var resp []func() datasource.DataSource

	resp = append(resp, core.DataSources()...)
	resp = append(resp, filestation.DataSources()...)
	resp = append(resp, virtualization.DataSources()...)
	resp = append(resp, container.DataSources()...)

	return resp
}

func (p *SynologyProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		NewISOFunction,
		NewMkPasswdFunction,
		NewIniEncodeFunction,
	}
}

func (p *SynologyProvider) ValidateConfig(
	ctx context.Context,
	req provider.ValidateConfigRequest,
	resp *provider.ValidateConfigResponse,
) {
	var data SynologyProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Host.IsNull() && !data.Host.IsUnknown() {
		if _, err := url.Parse(data.Host.ValueString()); err != nil {
			resp.Diagnostics.Append(
				diag.NewAttributeErrorDiagnostic(
					path.Root("host"),
					"invalid provider configuration",
					"host is not a valid URL"),
			)
			return
		}
	}

	// Validate OTP secret only if provided
	if !data.OtpSecret.IsNull() && !data.OtpSecret.IsUnknown() &&
		data.OtpSecret.ValueString() != "" {
		if err := validateOtpSecret(data.OtpSecret.ValueString()); err != nil {
			resp.Diagnostics.Append(
				diag.NewAttributeErrorDiagnostic(
					path.Root("otp_secret"),
					"Invalid OTP secret",
					err.Error(),
				),
			)
		}
	}

	// Validate session_cache configuration only if provided
	if !data.SessionCache.IsNull() && !data.SessionCache.IsUnknown() {
		var sessionCache SessionCacheModel
		diags := data.SessionCache.As(
			ctx,
			&sessionCache,
			basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true},
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		mode := ""
		if !sessionCache.Mode.IsNull() && !sessionCache.Mode.IsUnknown() {
			mode = sessionCache.Mode.ValueString()
			if !isValidSessionCacheMode(mode) {
				resp.Diagnostics.Append(
					diag.NewAttributeErrorDiagnostic(
						path.Root("session_cache").AtName("mode"),
						"Invalid session cache mode",
						"session_cache.mode must be one of: 'auto', 'keyring', 'file', 'memory', 'off'",
					),
				)
			}
		}

		pstr := ""
		if !sessionCache.Path.IsNull() && !sessionCache.Path.IsUnknown() {
			pstr = sessionCache.Path.ValueString()
		}

		if mode == "file" && pstr == "" {
			resp.Diagnostics.Append(
				diag.NewAttributeWarningDiagnostic(
					path.Root("session_cache").AtName("path"),
					"Missing session_cache.path",
					"When session_cache.mode is \"file\", session_cache.path should be set to a writable directory. Will use default OS cache directory.",
				),
			)
		}
		if mode != "" && mode != "file" && pstr != "" {
			resp.Diagnostics.Append(
				diag.NewAttributeWarningDiagnostic(
					path.Root("session_cache").AtName("path"),
					"session_cache.path ignored",
					fmt.Sprintf(
						"session_cache.mode is %q, so session_cache.path will be ignored. Set mode = \"file\" to use it.",
						mode,
					),
				),
			)
		}
	}
}

func isValidSessionCacheMode(s string) bool {
	return s == "auto" || s == "keyring" || s == "file" || s == "memory" || s == "off"
}

func validateOtpSecret(s string) error {
	if len(s) < 16 || len(s) > 32 {
		return fmt.Errorf("invalid OTP secret length")
	}
	if !reBase32Chars.MatchString(s) {
		return fmt.Errorf("invalid OTP secret format")
	}
	return nil
}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &SynologyProvider{}
	}
}
