---
page_title: "Virtualization: synology_virtualization_guest"
subcategory: "Virtualization"
description: |-
  Guest data source
---

# Virtualization: Guest (Data Source)

Guest data source



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the guest.

### Read-Only

- `autorun` (Number) Hostname of Synology station.
- `description` (String) Description of the guest.
- `disks` (Set of Object) List of virtual disks. (see [below for nested schema](#nestedatt--disks))
- `id` (String) Unique identifier for this data source.
- `networks` (Set of Object) List of networks. (see [below for nested schema](#nestedatt--networks))
- `status` (String) Status of the guest.
- `storage_id` (String) Storage ID of the guest.
- `storage_name` (String) Storage name of the guest.
- `vcpu_num` (Number) Number of virtual CPUs.
- `vram_size` (Number) Size of virtual RAM.

<a id="nestedatt--disks"></a>
### Nested Schema for `disks`

Read-Only:

- `controller` (Number)
- `id` (String)
- `size` (Number)
- `unmap` (Boolean)


<a id="nestedatt--networks"></a>
### Nested Schema for `networks`

Read-Only:

- `id` (String)
- `mac` (String)
- `model` (Number)
- `name` (String)
- `vnic_id` (String)