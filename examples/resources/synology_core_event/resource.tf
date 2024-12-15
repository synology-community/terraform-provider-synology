resource "synology_core_event" "test" {
  name = "Test"

  script = "echo 'Hello, World!'"
  user   = "root"

  run  = true
  when = "apply"
}
