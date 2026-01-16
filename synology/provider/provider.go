package provider

import (
	"context"
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/99designs/keyring"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
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
		MarkdownDescription: "The Synology provider enables Terraform to manage resources on Synology DiskStation Manager (DSM) systems.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "Remote Synology URL, e.g. `https://host:5001`. Can be specified with the `SYNOLOGY_HOST` " +
					"environment variable.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						reHostname,
						"must be a valid URL",
					),
				},
			},
			"user": schema.StringAttribute{
				MarkdownDescription: "User to connect to Synology station with. Can be specified with the `SYNOLOGY_USER` " +
					"environment variable.",
				Optional: true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password to use when connecting to Synology station. Can be specified with the `SYNOLOGY_PASSWORD` " +
					"environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"otp_secret": schema.StringAttribute{
				MarkdownDescription: "OTP secret to use when connecting to Synology DSM. Can be specified with the `SYNOLOGY_OTP_SECRET` " +
					"environment variable. Format: Valid RFC 4648 base32 TOTP secret: A-Z, 2-7, optional '=', spaces ignored.",
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(16, 32),
					stringvalidator.RegexMatches(
						reBase32Chars,
						"must be base32 encoded secret only, not whole otp:// URI",
					),
				},
			},
			"skip_cert_check": schema.BoolAttribute{
				MarkdownDescription: "Whether to skip SSL certificate checks. Can be specified with the `SYNOLOGY_SKIP_CERT_CHECK` " +
					"environment variable.",
				Optional: true,
			},
			"session_cache": schema.SingleNestedAttribute{
				MarkdownDescription: "Session cache configuration. Supports caching Synology DSM sessions to reduce login frequency.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						MarkdownDescription: "Session cache mode - one of: auto, keyring, file, memory, off. Default: off. Can be set via `SYNOLOGY_SESSION_CACHE` environment variable.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("auto", "keyring", "file", "memory", "off"),
						},
					},
					"path": schema.StringAttribute{
						MarkdownDescription: "Directory for file-based session cache when mode = \"file\". Defaults to OS user cache dir. Can be set via `SYNOLOGY_SESSION_CACHE_PATH` environment variable.",
						Optional:            true,
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

	// Extract session cache configuration
	var sessionCache SessionCacheModel
	resp.Diagnostics.Append(getSessionConfig(ctx, data.SessionCache, &sessionCache)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var key, cacheK string
	key = sessionKey(host, user, skipCertificateCheck)
	var ring keyring.Keyring
	if sessionCache.Mode.ValueString() != "off" {
		cacheK = cacheKey(host, user, skipCertificateCheck)
		ring, _, _ = openSessionRing(
			sessionCache.Mode.ValueString(),
			sessionCache.Path.ValueString(),
		)
	}

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

	if sessionCache.Mode.ValueString() != "off" {
		if ring != nil {
			if rec, err := readSession(ring, cacheK); err == nil && rec != nil &&
				rec.SessionID != "" &&
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
		if sessionCache.Mode.ValueString() != "off" {
			if c, ok := cli.(*client.Client); ok && ring != nil {
				s := c.ExportSession()
				_ = writeSession(ring, cacheK, sessionRecord{
					SessionID:    s.SessionID,
					SynoToken:    s.SynoToken,
					IssuedAt:     s.CreatedAt,
					LastTotpStep: time.Now().Unix() / 30,
				})
			}
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
		resp.ActionData = c
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
		resp.ActionData = c
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

func (p *SynologyProvider) Actions(ctx context.Context) []func() action.Action {
	var resp []func() action.Action
	resp = append(resp, container.Actions()...)
	return resp
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
