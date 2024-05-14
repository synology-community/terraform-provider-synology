resource "synology_virtualization_guest" "foo" {
  name         = "testvm"
  storage_name = "default"

  vcpu_num  = 4
  vram_size = 4096

  network {
    name = "default"
  }

  disk {
    create_type = 0
    size        = 20000
  }
}
