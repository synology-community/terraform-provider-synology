---
page_title: "synology_container Resource - synology"
subcategory: "Container"
description: |-
  Manages a standalone Container Manager container.
---

# synology_container (Resource)

Manages a standalone Container Manager container. DSM has no in-place update for container profiles, so configuration changes force replacement.

Start, stop, and restart are available as the provider action (`synology_container_operation`), not attributes on this resource.

> **Delete limitation:** On the tested DSM 7.3 build, `SYNO.Docker.Container` method `delete` returns error `114` (`error_invalid`) for every documented parameter shape, including the community Python client's. Create, list, get, start, and stop work. Until the delete shape is identified, destroy may leave an orphan that must be removed in the Container Manager UI.

## Example Usage

```terraform
resource "synology_container" "hello" {
  name          = "tofu-hello"
  image         = "busybox:latest"
  run_instantly = false
  network       = ["bridge"]
}
```

## Schema

### Required

- `name` (String) Container name. Forces replacement.
- `image` (String) Image reference. Must exist on the NAS or be pullable. Forces replacement.

### Optional

- `cmd` (String) Command string.
- `privileged` (Boolean) Run privileged.
- `use_host_network` (Boolean) Use the host network stack.
- `enable_restart_policy` (Boolean) Enable restart policy.
- `cpu_priority` (Number) CPU priority (DSM scale).
- `memory_limit` (Number) Memory limit in bytes (0 = unlimited).
- `network` (List of String) Network names to attach.
- `env` (Map of String) Environment variables.
- `run_instantly` (Boolean) Start immediately after create.

### Read-Only

- `status` (String) Last observed status from DSM.

## Import

```console
tofu import synology_container.hello tofu-hello
```
