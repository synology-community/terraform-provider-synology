// basic usage
provider "synology" {
  host            = "https://synology.local:5001"
  user            = "tf-user"
  password        = "testing-password" // use it only for testing purposes, use SYNOLOGY_PASSWORD env var to set this value
  skip_cert_check = true               // use it only for testing purposes
}

// OTP with cached session in keyring
provider "synology" {
  host          = "https://synology.local:5001"
  user          = "tf-user"
  password      = "testing-password"
  otp_secret    = "0123456789ABCDEF"
  session_cache = "keyring" # auto | keyring | file | memory | off
  # If you use file mode:
  # session_cache     = "file"
  # session_cache_path = pathexpand("~/.cache/synology-tf")
}

// OTP with cached session in file
provider "synology" {
  host               = "https://synology.local:5001"
  user               = "tf-user"
  password           = "testing-password"
  otp_secret         = "0123456789ABCDEF"
  session_cache      = "file" # auto | keyring | file | memory | off
  session_cache_path = pathexpand("~/.cache/synology-tf")
}
