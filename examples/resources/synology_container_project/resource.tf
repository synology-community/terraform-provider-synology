resource "synology_container_project" "foo" {
  name = "foo"

  services = {
    "bar" = {
      image = {
        name = "nginx"
        tag  = "latest"
      }

      ports = [{
        target    = 80
        published = 80
      }]

      configs = [
        {
          source = "baz"
          target = "/config/baz.txt"
          gid    = 0
          uid    = 0
          mode   = "0660"
        },
        {
          source = "qux"
          target = "/config/qux.toml"
        }
      ]

      logging = { driver = "json-file" }
    }
  }

  configs = {
    "baz" = {
      content = "Hello, World!"
    }

    "qux" = {
      file = "/volume1/foo/bar"
    }
  }
}
