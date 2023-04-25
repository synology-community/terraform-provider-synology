data "synology_filestation_info" "filestation_info" {}

output "filestation_hostname" {
  value = data.synology_filestation_info.filestation_info.hostname
}
