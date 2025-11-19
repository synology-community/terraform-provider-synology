provider "synology" {
  host       = "https://nas.example.com:5001"
  user       = "admin"
  password   = "your-password"
  otp_secret = "YOUR_BASE32_TOTP_SECRET"
}
