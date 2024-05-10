//go:build tools

package tools

import (
	// Documentation generation
	_ "github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen"
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
	_ "golang.org/x/tools/cmd/stringer"
	- "github.com/rjeczalik/interfaces/cmd/structer"
)
