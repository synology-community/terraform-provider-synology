resource "synology_container_project" "default" {
  name = "foo"

  networks = {
    foo = {
      driver = "bridge"
    }

    bar = {
      driver = "macvlan"
      driver_opts = {
        "parent" = "ovs_bond0"
      }
      ipam = {
        driver = "macvlan"
        config = [{
          subnet   = "10.0.0.0/16"
          gateway  = "10.0.0.1"
          ip_range = "10.0.60.1/28"
          aux_address = {
            host = "10.0.60.2"
          }
        }]
      }
    }
  }
  configs = {
    foo = {
      name    = "foo.txt"
      content = "Hello World"
    }
  }
  services = {
    bar = {
      name     = "bar"
      replicas = 1

      image = "nginx"

      logging = {
        driver = "json-file"
      }

      configs = [
        {
          source = "foo"
          target = "/etc/foo.txt"
          mode   = "777"
        }
      ]

      ports = [{
        target    = 80
        published = "8557"
        protocol  = "tcp"
      }]

      networks = {
        foo = {}
      }
    }
  }
}
