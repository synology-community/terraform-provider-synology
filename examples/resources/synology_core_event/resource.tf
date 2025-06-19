resource "synology_core_event" "test" {
  name   = "Test"
  run    = true
  script = "echo 'Hello, World!'"
  user   = "root"
  when   = ["apply", "destroy", "upgrade"]
}
