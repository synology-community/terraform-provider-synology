# Terraform Synology Provider

This provider allows [Terraform](https://www.terraform.io/) to manage remote [Synology NAS](https://www.synology.com/dsm/solution/what-is-nas/for-home) server.

This repo uses the Synology [API client](https://www.github.com/synology-community/go-synology) to access remote NAS from Go code. Token caching (critically required for TOTP) provided by [99designs/keyring](https://github.com/99designs/keyring).

## Documentation Links

- [Synology Virtual Machine Manager](https://global.download.synology.com/download/Document/Software/DeveloperGuide/Package/Virtualization/All/enu/Synology_Virtual_Machine_Manager_API_Guide.pdf)
- [Synology File Station](https://global.download.synology.com/download/Document/Software/DeveloperGuide/Package/FileStation/All/enu/Synology_File_Station_API_Guide.pdf)

## Usage Example

The following example shows how to use `synology_api` to manage machine learning compute resource.

```hcl
terraform {
  required_providers {
    synology = {
      source  = "synology-community/synology"
    }
  }
}

provider "synology" {
  # More information on the authentication methods supported by
  # the AzApi Provider can be found here:
  # https://registry.terraform.io/providers/synology-community/synology/latest/docs

  # subscription_id = "..."
  # client_id       = "..."
  # client_secret   = "..."
  # tenant_id       = "..."
}

resource "synology_api" "foo" {
  api        = "SYNO.Core.System"
  method     = "info"
  version    = 1
  parameters = {
    "query" = "all"
  }
}
```

## Permissions required

### docker

- member of Administrators group (Container Manager has no RBAC)
- RW on target share
- applications granted:
  - DSM
  - File Station
