package modifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SetFilePathFromContent returns a plan modifier that sets the file path
// based on the config name when content is populated.
func SetFilePathFromContent() planmodifier.String {
	return setConfigFilePathFromContent{}
}

// setConfigFilePathFromContent implements the plan modifier.
type setConfigFilePathFromContent struct{}

// Description returns a human-readable description of the plan modifier.
func (m setConfigFilePathFromContent) Description(_ context.Context) string {
	return "Sets the file path from the config name when content is populated."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m setConfigFilePathFromContent) MarkdownDescription(_ context.Context) string {
	return "Sets the file path from the config name when content is populated."
}

// PlanModifyString implements the plan modification logic.
func (m setConfigFilePathFromContent) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	// Do nothing if there is an unknown configuration value
	if req.ConfigValue.IsUnknown() {
		return
	}

	// If file is already set, don't modify it
	if !req.ConfigValue.IsNull() && req.ConfigValue.ValueString() != "" {
		return
	}

	// Navigate up to the parent object to get the config attributes
	parentPath := req.Path.ParentPath()

	// Get the content attribute from the same config object
	contentPath := parentPath.AtName("content")
	var contentValue types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, contentPath, &contentValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the name attribute from the same config object
	namePath := parentPath.AtName("name")
	var nameValue types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, namePath, &nameValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If content is populated and name is available, set file to name
	if !contentValue.IsNull() && !contentValue.IsUnknown() && contentValue.ValueString() != "" &&
		!nameValue.IsNull() && !nameValue.IsUnknown() && nameValue.ValueString() != "" {
		resp.PlanValue = types.StringValue(nameValue.ValueString())
	}
}
