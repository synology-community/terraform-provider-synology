provider "synology" {
  host            = "synology.local:5001"
  user            = "tf-user"
  password        = "testing-password" // use it only for testing purposes, use SYNOLOGY_PASSWORD env var to set this value
  skip_cert_check = true               // use it only for testing purposes
}
