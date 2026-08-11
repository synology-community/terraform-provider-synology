---
page_title: "synology_container_registry Resource - synology"
subcategory: "Container"
description: |-
  Manages a Container Manager registry entry.
---

# synology_container_registry (Resource)

Manages a Container Manager registry entry. DSM exposes create and delete for third-party registries; `set` rejected documented parameter shapes on the tested DSM, so every attribute forces replacement.

Built-in Synology registries (`syno = true`) cannot be deleted.

## Example Usage

```terraform
resource "synology_container_registry" "ghcr" {
  name             = "ghcr"
  url              = "https://ghcr.io"
  enable_trust_ssc = false
}
```

## Schema

### Required

- `name` (String) Registry display name. Forces replacement.
- `url` (String) Registry base URL. Forces replacement.

### Optional

- `enable_trust_ssc` (Boolean) Trust self-signed certificates. Forces replacement.

### Read-Only

- `syno` (Boolean) True when this is a built-in Synology registry.

## Import

```console
tofu import synology_container_registry.ghcr ghcr
```
