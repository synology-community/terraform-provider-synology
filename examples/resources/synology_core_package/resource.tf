resource "synology_core_package" "mariadb" {
  name = "MariaDB10"

  wizard = {
    port              = 3306
    new_root_password = "T3stP@ssw0rd"
  }
}

# `run` is the only attribute that can be changed in place: toggling it starts
# or stops the package through SYNO.Core.Package.Control. Every other attribute
# forces replacement, which uninstalls the package first -- and uninstalling a
# package can remove the data it owns.
resource "synology_core_package" "container_manager" {
  name = "ContainerManager"
  run  = false
}
