resource "synology_api" "foo" {
  api     = "SYNO.Core.System"
  method  = "info"
  version = 1
  parameters = {
    "query" = "all"
  }
}