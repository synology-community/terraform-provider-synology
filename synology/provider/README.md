The Synology provider enables Terraform to manage resources on Synology DiskStation Manager (DSM) systems.

## Authentication

The provider supports multiple authentication methods, prioritizing explicit configuration over environment variables:

1. **Provider Configuration**: Set credentials directly in the provider block
1. **Environment Variables**: Use `SYNOLOGY_HOST`, `SYNOLOGY_USER`, and `SYNOLOGY_PASSWORD`

### Basic Authentication

```hcl
provider "synology" {
  host     = "https://nas.example.com:5001"
  user     = "admin"
  password = "your-password"
}
```

### With Two-Factor Authentication (OTP)

```hcl
provider "synology" {
  host       = "https://nas.example.com:5001"
  user       = "admin"
  password   = "your-password"
  otp_secret = "YOUR_BASE32_TOTP_SECRET"
}
```

### Using Environment Variables

```hcl
provider "synology" {
  # Configuration will be read from environment variables:
  # - SYNOLOGY_HOST
  # - SYNOLOGY_USER
  # - SYNOLOGY_PASSWORD
  # - SYNOLOGY_OTP_SECRET (optional)
  # - SYNOLOGY_SKIP_CERT_CHECK (optional)
  # - SYNOLOGY_SESSION_CACHE (optional)
}
```

```bash
export SYNOLOGY_HOST="https://nas.example.com:5001"
export SYNOLOGY_USER="admin"
export SYNOLOGY_PASSWORD="your-password"
terraform plan
```

## Session Caching

The provider supports session caching to avoid repeated authentication:

- **`auto`** (default): Tries keyring backends, falls back to file/memory
- **`keyring`**: Uses OS keyring (Keychain, WinCred, SecretService, KWallet)
- **`file`**: Stores sessions in a file (requires `session_cache_path`)
- **`memory`**: In-memory cache (lost on provider restart)
- **`off`**: No caching (authenticate every time)

### Example with File-based Cache

```hcl
provider "synology" {
  host               = "https://nas.example.com:5001"
  user               = "admin"
  password           = "your-password"
  session_cache      = "file"
  session_cache_path = "/tmp/terraform-synology-sessions"
}
```

## SSL Certificate Verification

By default, the provider skips SSL certificate verification (useful for self-signed certificates). To enable verification:

```hcl
provider "synology" {
  host            = "https://nas.example.com:5001"
  user            = "admin"
  password        = "your-password"
  skip_cert_check = false
}
```

## Supported Resources

The provider supports managing:

- **Container Projects**: Docker Compose projects and containers
- **Virtual Machines**: Virtualization guests and images
- **File Station**: Files, folders, and ISO images
- **Core System**: Packages, tasks, and events
- **Network Configuration**: Network interfaces and settings

## API Rate Limiting

The provider includes automatic retry logic with exponential backoff for API rate limiting. The default retry limit is 5 attempts.

## Notes

- The provider requires DSM 7.0 or later
- Some resources require specific DSM packages to be installed
- Two-factor authentication requires a valid RFC 4648 base32 TOTP secret (16-32 characters, A-Z and 2-7)
- Session caching significantly reduces authentication overhead for repeated operations
