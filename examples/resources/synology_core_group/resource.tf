resource "synology_core_group" "platform" {
  name        = "platform-ops"
  description = "Operators for the engineering platform"
}

# Membership is declared on synology_core_user.groups, not here.
