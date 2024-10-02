package models

import (
	"context"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ComposeContent struct {
	*types.String
}

type ComposeContentBuilder struct {
	project composetypes.Project
	diags   diag.Diagnostics
	ctx     context.Context
}

func NewComposeContentBuilder(ctx context.Context) *ComposeContentBuilder {
	return &ComposeContentBuilder{
		project: composetypes.Project{},
		diags:   diag.Diagnostics{},
		ctx:     ctx,
	}
}

func (c *ComposeContentBuilder) SetNetworks(networks *types.Set) *ComposeContentBuilder {
	if !networks.IsNull() && !networks.IsUnknown() {

		elements := []Network{}
		c.diags.Append(networks.ElementsAs(c.ctx, &elements, true)...)

		if c.diags.HasError() {
			return c
		}

		c.project.Networks = map[string]composetypes.NetworkConfig{}

		for _, v := range elements {
			n := composetypes.NetworkConfig{}

			c.diags.Append(v.AsComposeConfig(c.ctx, &n)...)
			if c.diags.HasError() {
				return c
			}

			c.project.Networks[n.Name] = n
		}
	}
	return c
}

func (c *ComposeContentBuilder) SetServices(services *types.Set) *ComposeContentBuilder {
	if !services.IsNull() && !services.IsUnknown() {

		elements := []Service{}
		c.diags.Append(services.ElementsAs(c.ctx, &elements, true)...)

		if c.diags.HasError() {
			return c
		}

		c.project.Services = map[string]composetypes.ServiceConfig{}

		for _, v := range elements {
			s := composetypes.ServiceConfig{}

			c.diags.Append(v.AsComposeConfig(c.ctx, &s)...)
			if c.diags.HasError() {
				return c
			}

			c.project.Services[s.Name] = s
		}
	}
	return c
}

func (c *ComposeContentBuilder) SetVolumes(volumes *types.Set) *ComposeContentBuilder {
	if !volumes.IsNull() && !volumes.IsUnknown() {

		elements := []Volume{}
		c.diags.Append(volumes.ElementsAs(c.ctx, &elements, true)...)

		if c.diags.HasError() {
			return c
		}

		c.project.Volumes = map[string]composetypes.VolumeConfig{}

		for _, v := range elements {
			vol := composetypes.VolumeConfig{}

			c.diags.Append(v.AsComposeConfig(c.ctx, &vol)...)
			if c.diags.HasError() {
				return c
			}

			c.project.Volumes[vol.Name] = vol
		}
	}
	return c
}

func (c *ComposeContentBuilder) SetConfigs(configs *types.Set) *ComposeContentBuilder {
	if !configs.IsNull() && !configs.IsUnknown() {

		elements := []Config{}
		c.diags.Append(configs.ElementsAs(c.ctx, &elements, true)...)

		if c.diags.HasError() {
			return c
		}

		c.project.Configs = map[string]composetypes.ConfigObjConfig{}

		for _, v := range elements {
			cfg := composetypes.ConfigObjConfig{}

			c.diags.Append(v.AsComposeConfig(c.ctx, &cfg)...)
			if c.diags.HasError() {
				return c
			}

			c.project.Configs[cfg.Name] = cfg
		}
	}
	return c
}

func (c *ComposeContentBuilder) Build(content *string) diag.Diagnostics {

	projectYAML, err := c.project.MarshalYAML()
	if err != nil {
		c.diags.Append(diag.NewErrorDiagnostic("Failed to marshal docker-compose.yml", err.Error()))
		return c.diags
	}
	pyaml := string(projectYAML)
	*content = pyaml

	return c.diags
}

// func (c *ComposeContent) UnmarshalJSON(data *) error {
// 	c.String = types.StringValue()
// 	c.string = string(data)
// 	return nil
// }
