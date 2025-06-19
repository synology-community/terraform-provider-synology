package models

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ProjectResourceModel represents the project resource model.
// This is a temporary reference - consider importing from parent package if needed.
// ProjectResourceModel describes the resource data model.
type ProjectResourceModel struct {
	ID            types.String      `tfsdk:"id"`
	Name          types.String      `tfsdk:"name"`
	SharePath     types.String      `tfsdk:"share_path"`
	Services      types.Map         `tfsdk:"services"`
	Networks      types.Map         `tfsdk:"networks"`
	Volumes       types.Map         `tfsdk:"volumes"`
	Secrets       types.Map         `tfsdk:"secrets"`
	Configs       types.Map         `tfsdk:"configs"`
	Extensions    types.Map         `tfsdk:"extensions"`
	Run           types.Bool        `tfsdk:"run"`
	Status        types.String      `tfsdk:"status"`
	ServicePortal types.Object      `tfsdk:"service_portal"`
	Content       types.String      `tfsdk:"content"`
	Metadata      types.Map         `tfsdk:"metadata"`
	CreatedAt     timetypes.RFC3339 `tfsdk:"created_at"`
	UpdatedAt     timetypes.RFC3339 `tfsdk:"updated_at"`
}

func (p ProjectResourceModel) IsRunning() bool {
	return strings.ToUpper(p.Status.ValueString()) == "RUNNING"
}

func (p ProjectResourceModel) ShouldRun() bool {
	return !p.IsRunning() && p.Run.ValueBool()
}

func (m ProjectResourceModel) ModelType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: m.AttrType(),
	}
}

// AttrType returns the attribute types for the ProjectResourceModel.
func (m ProjectResourceModel) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"id":             types.StringType,
		"name":           types.StringType,
		"share_path":     types.StringType,
		"services":       types.MapType{ElemType: Service{}.ModelType()},
		"networks":       types.MapType{ElemType: Network{}.ModelType()},
		"volumes":        types.MapType{ElemType: Volume{}.ModelType()},
		"secrets":        types.MapType{ElemType: Secret{}.ModelType()},
		"configs":        types.MapType{ElemType: Config{}.ModelType()},
		"extensions":     types.MapType{ElemType: Extension{}.ModelType()},
		"run":            types.BoolType,
		"status":         types.StringType,
		"service_portal": ServicePortal{}.ModelType(),
		"content":        types.StringType,
		"metadata":       types.MapType{ElemType: types.StringType},
	}
}

func (m ProjectResourceModel) ConfigRaw(
	ctx context.Context,
	yamlContent *string,
) diag.Diagnostics {
	// Convert the config to yaml content
	return NewComposeContentBuilder(
		ctx,
	).SetServices(
		&m.Services,
	).SetNetworks(
		&m.Networks,
	).SetVolumes(
		&m.Volumes,
	).SetConfigs(
		&m.Configs,
	).SetSecrets(
		&m.Secrets,
	).Build(
		yamlContent,
	)
}
