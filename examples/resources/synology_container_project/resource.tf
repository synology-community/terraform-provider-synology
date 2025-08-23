resource "synology_container_project" "foo" {
  name       = "foo"
  share_path = "/docker/foo"
  run        = true

  services = {
    "bar" = {
      image = "nginx:latest"

      ports = [{
        target    = 80
        published = 8888
      }]

      configs = [
        {
          name   = "index"
          source = "index"
          target = "/usr/share/nginx/html/index.html"
          gid    = 0
          uid    = 0
          mode   = "0660"
        },
        {
          name   = "compose"
          source = "compose"
          target = "/usr/share/nginx/html/compose.yaml"
        }
      ]

      logging = { driver = "json-file" }
    }
  }

  configs = {
    "index" = {
      name    = "index"
      content = "<h1>Hello, World!!!</h1><a href=\"/compose.yaml\">compose.yaml</a>"
    }

    "compose" = {
      name = "compose"
      file = "/volume1/docker/foo/compose.yaml"
    }
  }
}
