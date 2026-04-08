package virtualization

import (
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/synology-community/go-synology/pkg/api/virtualization"
)

// resolveStorage resolves the storage to use. It checks storageID first,
// then storageName, and falls back to the first available storage.
func resolveStorage(
	storages []virtualization.Storage,
	storageID string,
	storageName string,
) (virtualization.Storage, diag.Diagnostics) {
	var diags diag.Diagnostics
	var match *virtualization.Storage

	if storageID != "" {
		i := slices.IndexFunc(storages, func(s virtualization.Storage) bool {
			return s.ID == storageID
		})
		if i != -1 {
			match = &storages[i]
		}
	}

	if match == nil && storageName != "" {
		i := slices.IndexFunc(storages, func(s virtualization.Storage) bool {
			return s.Name == storageName
		})
		if i != -1 {
			match = &storages[i]
		}
	}

	if match == nil && len(storages) > 0 {
		match = &storages[0]
	}

	if match == nil {
		diags.AddError(
			"Storage not found",
			"Unable to find storage. Specify storage_id or storage_name, or ensure VMM has configured storages.",
		)
		return virtualization.Storage{}, diags
	}

	return *match, diags
}
