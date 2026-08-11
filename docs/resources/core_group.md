---
page_title: "synology_core_group Resource - synology"
subcategory: "Core"
description: |-
  Manages a DSM local group. Membership lives on synology_core_user.groups.
---

# synology_core_group (Resource)

Manages a DSM local group. Membership is declared on `synology_core_user.groups`, not here.

## Example Usage

```terraform
resource "synology_core_group" "platform" {
  name        = "platform-ops"
  description = "Operators for the engineering platform"
}
```

## Schema

### Required

- `name` (String) Group name. Changing this forces replacement.

### Optional

- `description` (String) Group description.

### Read-Only

- `gid` (Number) Numeric group id assigned by DSM.

## Import

```console
tofu import synology_core_group.platform platform-ops
```
