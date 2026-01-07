terraform {
  required_providers {
    synology = {
      source = "synology-community/synology"
    }
  }
}

provider "synology" {
  # host     = "nas.example.com"
  # port     = 5001
  # username = "admin"
  # password = var.synology_password
}

# Example: Start a container
action "synology_container_operation" "start_container" {
  config {
    name      = "tinkerbell"
    operation = "start"
  }
}

# Example: Stop a container
action "synology_container_operation" "stop_container" {
  config {
    name      = "my-app"
    operation = "stop"
  }
}

# Example: Restart a container
action "synology_container_operation" "restart_container" {
  config {
    name      = "web-server"
    operation = "restart"
  }
}

# Example: Using terraform_data with action_triggers
# This resource can be used to trigger container operations on changes
resource "terraform_data" "container_restart_trigger" {
  input = timestamp()

  lifecycle {
    action_trigger {
      events  = [after_create, after_update]
      actions = [action.synology_container_operation.restart_container]
    }
  }
}

# Example: Trigger multiple container operations in sequence
resource "terraform_data" "multi_container_trigger" {
  input = "v1.0.0"

  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [
        action.synology_container_operation.stop_container,
        action.synology_container_operation.start_container
      ]
    }
  }
}

# To invoke an action from the CLI:
# terraform apply -invoke=action.synology_container_operation.start_container
# terraform apply -invoke=action.synology_container_operation.stop_container
# terraform apply -invoke=action.synology_container_operation.restart_container

# Example: Trigger an action after creating a container resource
# resource "synology_container_project" "example" {
#   name = "my-project"
#   # ... other configuration
#
#   lifecycle {
#     action_trigger {
#       events  = [after_create, after_update]
#       actions = [action.synology_container_operation.restart_container]
#     }
#   }
# }


