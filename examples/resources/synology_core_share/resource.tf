resource "synology_core_share" "scratch" {
  name     = "tofu-scratch"
  vol_path = "/volume1"
  desc     = "Scratch share managed by OpenTofu"

  enable_recycle_bin     = true
  recycle_bin_admin_only = true
  enable_share_compress  = false
  hidden                 = false
}
