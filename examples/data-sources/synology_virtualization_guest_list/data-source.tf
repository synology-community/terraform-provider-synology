data "synology_virtualization_guest_list" "all" {}

output "all_guests" {
  value = data.synology_virtualization_guest_list.all.guests
}
