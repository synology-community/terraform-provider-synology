resource "synology_container_registry" "ghcr" {
  name             = "ghcr"
  url              = "https://ghcr.io"
  enable_trust_ssc = false
}
