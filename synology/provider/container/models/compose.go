package models

import (
	"context"
	"regexp"

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

func (c *ComposeContentBuilder) SetNetworks(networks *types.Map) *ComposeContentBuilder {
	if !networks.IsNull() && !networks.IsUnknown() {

		elements := map[string]Network{}
		c.diags.Append(networks.ElementsAs(c.ctx, &elements, true)...)

		if c.diags.HasError() {
			return c
		}

		c.project.Networks = map[string]composetypes.NetworkConfig{}

		for k, v := range elements {
			n := composetypes.NetworkConfig{}

			c.diags.Append(v.AsComposeConfig(c.ctx, &n)...)
			if c.diags.HasError() {
				return c
			}

			c.project.Networks[k] = n
		}
	}
	return c
}

func (c *ComposeContentBuilder) SetServices(services *types.Map) *ComposeContentBuilder {
	if !services.IsNull() && !services.IsUnknown() {

		elements := map[string]Service{}
		c.diags.Append(services.ElementsAs(c.ctx, &elements, true)...)

		if c.diags.HasError() {
			return c
		}

		c.project.Services = map[string]composetypes.ServiceConfig{}

		for k, v := range elements {
			s := composetypes.ServiceConfig{}

			c.diags.Append(v.AsComposeConfig(c.ctx, &s)...)
			if c.diags.HasError() {
				return c
			}

			c.project.Services[k] = s
		}
	}
	return c
}

func (c *ComposeContentBuilder) SetVolumes(volumes *types.Map) *ComposeContentBuilder {
	if !volumes.IsNull() && !volumes.IsUnknown() {

		elements := map[string]Volume{}
		c.diags.Append(volumes.ElementsAs(c.ctx, &elements, true)...)

		if c.diags.HasError() {
			return c
		}

		c.project.Volumes = map[string]composetypes.VolumeConfig{}

		for k, v := range elements {
			vol := composetypes.VolumeConfig{}

			c.diags.Append(v.AsComposeConfig(c.ctx, &vol)...)
			if c.diags.HasError() {
				return c
			}

			c.project.Volumes[k] = vol
		}
	}
	return c
}

func (c *ComposeContentBuilder) SetConfigs(configs *types.Map) *ComposeContentBuilder {
	if !configs.IsNull() && !configs.IsUnknown() {

		elements := map[string]Config{}
		c.diags.Append(configs.ElementsAs(c.ctx, &elements, true)...)

		if c.diags.HasError() {
			return c
		}

		c.project.Configs = map[string]composetypes.ConfigObjConfig{}

		for k, v := range elements {
			cfg := composetypes.ConfigObjConfig{}

			c.diags.Append(v.AsComposeConfig(c.ctx, &cfg)...)
			if c.diags.HasError() {
				return c
			}

			c.project.Configs[k] = cfg
		}
	}
	return c
}

func (c *ComposeContentBuilder) SetSecrets(secrets *types.Map) *ComposeContentBuilder {
	if !secrets.IsNull() && !secrets.IsUnknown() {

		elements := map[string]Secret{}
		c.diags.Append(secrets.ElementsAs(c.ctx, &elements, true)...)

		if c.diags.HasError() {
			return c
		}

		c.project.Secrets = map[string]composetypes.SecretConfig{}

		for k, v := range elements {
			sec := composetypes.SecretConfig{}

			c.diags.Append(v.AsComposeConfig(c.ctx, &sec)...)
			if c.diags.HasError() {
				return c
			}

			c.project.Secrets[k] = sec
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

	re := regexp.MustCompile("\n[ ]+required: (true|false)\n")
	pycp := re.ReplaceAllString(pyaml, "\n")

	*content = pycp

	return c.diags
}

// func (c *ComposeContent) UnmarshalJSON(data *) error {
// 	c.String = types.StringValue()
// 	c.string = string(data)
// 	return nil
// }
