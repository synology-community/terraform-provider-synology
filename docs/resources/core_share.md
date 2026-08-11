---
page_title: "synology_core_share Resource - synology"
subcategory: "Core"
description: |-
  Manages a DSM shared folder with in-place updates via SYNO.Core.Share set.
---

# synology_core_share (Resource)

Manages a DSM shared folder. Name and volume force replacement; description, visibility, recycle bin, and compression update in place via `SYNO.Core.Share` method `set` (probed on DSM 7.3).

## Example Usage

```terraform
resource "synology_core_share" "scratch" {
  name     = "tofu-scratch"
  vol_path = "/volume1"
  desc     = "Scratch share managed by OpenTofu"

  enable_recycle_bin     = true
  recycle_bin_admin_only = true
  enable_share_compress  = false
  hidden                 = false
}
```

## Schema

### Required

- `name` (String) Share name. Changing this forces replacement.
- `vol_path` (String) Volume path, e.g. `/volume1`. Changing this forces replacement.

### Optional

- `desc` (String) Share description.
- `hidden` (Boolean) Hide the share from browse lists.
- `enable_recycle_bin` (Boolean) Enable the share recycle bin.
- `recycle_bin_admin_only` (Boolean) Restrict recycle bin access to administrators.
- `enable_share_compress` (Boolean) Enable Btrfs compression when supported.

### Read-Only

- `uuid` (String) DSM-assigned share UUID.

## Import

```console
tofu import synology_core_share.scratch tofu-scratch
```
