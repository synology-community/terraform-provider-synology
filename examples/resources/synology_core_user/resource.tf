terraform {
  required_version = ">= 1.11.0"

  required_providers {
    synology = {
      source = "synology-community/synology"
    }
  }
}

# password_wo is write-only: never stored in state. Bump password_wo_version to rotate.
resource "synology_core_user" "svc" {
  name                = "tofu-svc"
  password_wo         = var.initial_password
  password_wo_version = 1
  description         = "OpenTofu-managed service account"
  email               = "ops@example.com"
  groups              = ["users"]
}

variable "initial_password" {
  type      = string
  sensitive = true
}
