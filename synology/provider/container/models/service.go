package models

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	composetypes "github.com/compose-spec/compose-go/v2/types"
)

type Logging struct {
	Driver  types.String `tfsdk:"driver"`
	Options types.Map    `tfsdk:"options"`
}

type Image struct {
	Name       types.String `tfsdk:"name"`
	Repository types.String `tfsdk:"repository"`
	Tag        types.String `tfsdk:"tag"`
}

type Service struct {
	Name     types.String `tfsdk:"name"`
	Image    types.Set    `tfsdk:"image"`
	Replicas types.Int64  `tfsdk:"replicas"`
	Logging  types.Set    `tfsdk:"logging"`
	// Ports    types.SetType `tfsdk:"ports"`
}

func (m Service) ModelType() attr.Type {
	return types.ObjectType{AttrTypes: m.AttrType()}
}

func (m Service) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"name": types.StringType,
		"image": types.SetType{
			ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":       types.StringType,
					"repository": types.StringType,
					"tag":        types.StringType,
				},
			},
		},
		"replicas": types.Int64Type,
		"logging": types.SetType{
			ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"driver": types.StringType,
					"options": types.MapType{
						ElemType: types.StringType,
					},
				},
			},
		},
	}
}

func (m Service) Value() attr.Value {

	var logging basetypes.SetValue
	var image basetypes.SetValue

	if l, diag := m.Logging.ToSetValue(context.Background()); !diag.HasError() {
		logging = l
	}

	if i, diag := m.Image.ToSetValue(context.Background()); !diag.HasError() {
		image = i
	}

	return types.ObjectValueMust(m.AttrType(), map[string]attr.Value{
		"name":     types.StringValue(m.Name.ValueString()),
		"image":    image,
		"replicas": types.Int64Value(m.Replicas.ValueInt64()),
		"logging":  logging,
	})
}

func (m Service) AsComposeServiceConfig() composetypes.ServiceConfig {
	service := composetypes.ServiceConfig{}

	sName := m.Name.ValueString()

	if !m.Logging.IsNull() && !m.Logging.IsUnknown() {
		service.Logging = &composetypes.LoggingConfig{}
		logging := []Logging{}
		if diag := m.Logging.ElementsAs(context.Background(), &logging, true); !diag.HasError() {
			service.Logging.Driver = logging[0].Driver.ValueString()

			opts := map[string]string{}
			if diag := logging[0].Options.ElementsAs(context.Background(), &opts, true); !diag.HasError() {
				service.Logging.Options = opts
			}
		}
	}

	if !m.Image.IsNull() && !m.Image.IsUnknown() {
		image := []Image{}
		if diag := m.Image.ElementsAs(context.Background(), &image, true); !diag.HasError() {
			i := image[0]
			iName := i.Name.ValueString()
			var iTag, iRepo string
			if i.Repository.IsNull() || i.Repository.IsUnknown() {
				iRepo = "docker.io"
			} else {
				iRepo = i.Repository.ValueString()
			}
			if i.Tag.IsNull() || i.Tag.IsUnknown() {
				iTag = "latest"
			} else {
				iTag = i.Tag.ValueString()
			}
			service.Image = fmt.Sprintf("%s/%s:%s", iRepo, iName, iTag)
		}
	}

	service.Name = sName
	replicas := m.Replicas.ValueInt64()
	intReplicas := int(replicas)
	service.Deploy = &composetypes.DeployConfig{
		Replicas: &intReplicas,
	}

	return service
}
