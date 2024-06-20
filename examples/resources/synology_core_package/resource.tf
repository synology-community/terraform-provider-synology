resource "synology_core_package" "mariadb" {
  name = "MariaDB10"

  wizard = {
    port              = 3306
    new_root_password = "T3stP@ssw0rd"
  }
}
