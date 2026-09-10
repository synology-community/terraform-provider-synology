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

# `volume_path` names the volume the package is installed onto. It is optional:
# when omitted, the volume is resolved from DSM's package settings -- its
# configured default if it has one, otherwise the NAS's only volume.
#
# Set it explicitly on a multi-volume NAS. With several volumes and no DSM
# default, omitting it is an error rather than an arbitrary choice: which volume
# a package lands on decides where its data lives, and DSM offers no way to move
# it afterwards. Changing it reinstalls the package for the same reason.
resource "synology_core_package" "plex" {
  name        = "Plex Media Server"
  volume_path = "/volume1"
}
