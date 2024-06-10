package models

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	composetypes "github.com/compose-spec/compose-go/v2/types"
)

type Service struct {
	Name     types.String `tfsdk:"name"`
	Image    types.String `tfsdk:"image"`
	Replicas types.Int64  `tfsdk:"replicas"`
	// Ports    types.SetType `tfsdk:"ports"`
}

func (m Service) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m Service) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"name":     types.StringType,
		"image":    types.StringType,
		"replicas": types.Int64Type,
	}
}

func (m Service) Value() attr.Value {
	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"name":     types.StringValue(m.Name.ValueString()),
		"image":    types.StringValue(m.Image.ValueString()),
		"replicas": types.Int64Value(m.Replicas.ValueInt64()),
	})
}

func (m Service) AsComposeServiceConfig() composetypes.ServiceConfig {
	service := composetypes.ServiceConfig{}

	sName := m.Name.ValueString()

	service.Name = sName
	service.Image = m.Image.ValueString()
	replicas := m.Replicas.ValueInt64()
	intReplicas := int(replicas)
	service.Deploy = &composetypes.DeployConfig{
		Replicas: &intReplicas,
	}

	return service
}
