package models

import (
	"github.com/synology-community/terraform-provider-synology/synology/models/composetypes"
)

// HostsList is a list of colon-separated host-ip mappings.
type HostsList composetypes.HostsList

func (hl *HostsList) MarshalYAML() (any, error) {
	h := composetypes.HostsList(*hl)
	return h.AsList(":"), nil
}
