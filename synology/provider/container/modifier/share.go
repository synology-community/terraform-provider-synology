package modifier

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type useDefaultSharePath struct{}

// Description implements planmodifier.String.
func (u useDefaultSharePath) Description(context.Context) string {
	return "If the share path is not set, it will be set to /docker/<name>."
}

// MarkdownDescription implements planmodifier.String.
func (u useDefaultSharePath) MarkdownDescription(context.Context) string {
	return "If the share path is not set, it will be set to /docker/<name>."
}

// PlanModifyString implements planmodifier.String.
func (u useDefaultSharePath) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	// Do nothing if there is an unknown configuration value, otherwise interpolation gets messed up.
	if req.ConfigValue.IsUnknown() {
		return
	}

	if !req.PlanValue.IsUnknown() {
		return
	}

	// Set the share path to /docker/<name> if it is not set.
	var name string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(AttrName), &name)...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddAttributeError(
			path.Root(AttrName),
			"failed to get name attribute during plan modification",
			"",
		)
		return
	}

	resp.PlanValue = types.StringValue(fmt.Sprintf("/docker/%s", name))
}

func UseDefaultSharePath() planmodifier.String {
	return useDefaultSharePath{}
}
