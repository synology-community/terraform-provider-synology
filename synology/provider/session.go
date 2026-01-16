package provider

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/99designs/keyring"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"golang.org/x/sync/singleflight"
)

// singleflight deduplicates concurrent logins within a single provider process.
var (
	sf singleflight.Group
)

func getSessionConfig(
	ctx context.Context,
	src types.Object,
	target *SessionCacheModel,
) diag.Diagnostics {
	var resp diag.Diagnostics

	if !src.IsNull() && !src.IsUnknown() {
		diags := src.As(
			ctx,
			target,
			basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true},
		)
		resp.Append(diags...)
		if resp.HasError() {
			return resp
		}
		target.Mode = types.StringValue(
			getStringWithEnvFallback(target.Mode, SYNOLOGY_SESSION_CACHE_MODE),
		)
		target.Path = types.StringValue(
			getStringWithEnvFallback(target.Path, SYNOLOGY_SESSION_CACHE_PATH),
		)
	} else {
		// Check environment variables if session_cache object is not provided
		if v := os.Getenv(SYNOLOGY_SESSION_CACHE_MODE); v != "" {
			target.Mode = types.StringValue(v)
		}
		if v := os.Getenv(SYNOLOGY_SESSION_CACHE_PATH); v != "" {
			target.Path = types.StringValue(v)
		}
	}

	// Use default if still empty
	if target.Mode.IsNull() || target.Mode.IsUnknown() || target.Mode.ValueString() == "" {
		target.Mode = types.StringValue("off")
	}

	// Validate cache mode
	if !isValidSessionCacheMode(target.Mode.ValueString()) {
		resp.Append(diag.NewAttributeErrorDiagnostic(
			path.Root("session_cache").AtName("mode"),
			"Invalid session cache mode",
			fmt.Sprintf(
				"Unsupported value %q; valid values are: auto, keyring, file, memory, off.",
				target.Mode.ValueString(),
			),
		))
	}

	return resp
}

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
