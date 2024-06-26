---
page_title: "synology_api Resource - synology"
subcategory: ""
description: |-
  A Generic API Resource for making calls to the Synology DSM API.
---

# Api: (Resource)

A Generic API Resource for making calls to the Synology DSM API.

## Example Usage

```terraform
resource "synology_api" "foo" {
  api     = "SYNO.Core.System"
  method  = "info"
  version = 1
  parameters = {
    "query" = "all"
  }
}

output "result" {
  value = synology_api.foo.result
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `api` (String) The API to invoke.
- `method` (String) The method to invoke.

### Optional

- `parameters` (Map of String) Name of the storage device.
- `version` (Number) The version of the API to invoke.
- `when` (String)

### Read-Only

- `result` (Dynamic) The result of the API call.