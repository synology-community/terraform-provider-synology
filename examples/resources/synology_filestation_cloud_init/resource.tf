resource "synology_filestation_cloud_init" "foo" {
  path           = "/data/foo/bar/test.iso"
  user_data      = "#cloud-config\n\nusers:\n  - name: test\n    groups: sudo\n    shell: /bin/bash\n    sudo: ['ALL=(ALL) NOPASSWD:ALL']\n    ssh_authorized_keys:\n      - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDf7"
  create_parents = true
  overwrite      = true
}
