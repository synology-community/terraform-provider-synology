package models

// import (
// 	"context"
// 	"log"
// 	"slices"

// 	"github.com/compose-spec/compose-go/v2/loader"
// 	"github.com/synology-community/terraform-provider-synology/synology/models/composetypes"
// 	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
// 	"github.com/hashicorp/terraform-plugin-framework/types"

// 	"github.com/hashicorp/terraform-plugin-framework/diag"
// )

// type Project struct {
// 	ID            types.String      `tfsdk:"id"`
// 	Name          types.String      `tfsdk:"name"`
// 	SharePath     types.String      `tfsdk:"share_path"`
// 	Services      types.Set         `tfsdk:"service"`
// 	Networks      types.Set         `tfsdk:"network"`
// 	Volumes       types.Set         `tfsdk:"volume"`
// 	Secrets       types.Set         `tfsdk:"secret"`
// 	Configs       types.Set         `tfsdk:"config"`
// 	Extensions    types.Set         `tfsdk:"extension"`
// 	Run           types.Bool        `tfsdk:"run"`
// 	Status        types.String      `tfsdk:"status"`
// 	ServicePortal types.Set         `tfsdk:"service_portal"`
// 	CreatedAt     timetypes.RFC3339 `tfsdk:"created_at"`
// 	UpdatedAt     timetypes.RFC3339 `tfsdk:"updated_at"`
// }

// func (p *Project) FromComposeConfig(yaml string) diag.Diagnostics {
// 	diags := diag.Diagnostics{}

// 	pr, err := loader.LoadWithContext(context.Background(), composetypes.ConfigDetails{
// 		ConfigFiles: []composetypes.ConfigFile{
// 			{
// 				Content: []byte(yaml),
// 			},
// 		},
// 	})

// 	if err != nil {
// 		log.Printf("error loading compose config: %v", err)
// 	}

// 	services := []Service{}
// 	if !p.Services.IsNull() && !p.Services.IsUnknown() {
// 		if diag := p.Services.ElementsAs(context.Background(), &services, true); diag.HasError() {
// 			diags = append(diags, diag...)
// 		}
// 	}

// 	for k, s := range pr.Services {
// 		i := slices.IndexFunc(services, func(s Service) bool {
// 			return s.Name.ValueString() == k
// 		})
// 		sv := Service{}
// 		if i != -1 {
// 			sv = services[i]
// 		}

// 		commands := []string{}
// 		if !sv.Command.IsNull() && !sv.Command.IsUnknown() {
// 			if diag := sv.Command.ElementsAs(context.Background(), &commands, true); diag.HasError() {
// 				diags = append(diags, diag...)
// 			}
// 		}

// 		for _, c := range commands {
// 			commands = append(commands, c)
// 		}
// 	}

// 	return diags
// }
