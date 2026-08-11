# Standalone container. Prefer synology_container_project for multi-service apps.
# The image must already exist on the NAS (or be pullable by Container Manager).
resource "synology_container" "hello" {
  name          = "tofu-hello"
  image         = "hello-world:latest"
  run_instantly = false
  network       = ["bridge"]
}
