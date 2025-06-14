package core

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api"
	"github.com/synology-community/go-synology/pkg/api/core"
)

type PackageResourceModel struct {
	Name    types.String `tfsdk:"name"`
	Version types.String `tfsdk:"version"`
	File    types.String `tfsdk:"file"`
	URL     types.String `tfsdk:"url"`
	Wizard  types.Map    `tfsdk:"wizard"`
	Beta    types.Bool   `tfsdk:"beta"`

	Run types.Bool `tfsdk:"run"`
}

var _ resource.Resource = &PackageResource{}

func NewPackageResource() resource.Resource {
	return &PackageResource{}
}

type PackageResource struct {
	client core.Api
}

// Create implements resource.Resource.
func (p *PackageResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data PackageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	size := int64(0)
	if data.URL.ValueString() == "" {
		pkg, err := p.client.PackageFind(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to find package", err.Error())
			return
		}

		if data.Version.IsUnknown() || data.Version.IsNull() {
			data.Version = types.StringValue(pkg.Version)
		}

		if data.URL.IsUnknown() || data.URL.IsNull() {
			data.URL = types.StringValue(pkg.Link)
		}

		if pkg.Size != 0 {
			size = pkg.Size
		}
	}

	if size == 0 {
		s, err := p.client.ContentLength(context.Background(), data.URL.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to get file size", err.Error())
			return
		}
		size = s
	}

	wizardConf := make(map[string]string)
	if !data.Wizard.IsNull() && !data.Wizard.IsUnknown() {
		data.Wizard.ElementsAs(ctx, &wizardConf, true)
	}

	err := p.client.PackageInstallCompound(ctx, core.PackageInstallCompoundRequest{
		Name:        data.Name.ValueString(),
		URL:         data.URL.ValueString(),
		Size:        size,
		ExtraValues: wizardConf,
		Run:         data.Run.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Package install failed", err.Error())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update implements resource.Resource.
func (p *PackageResource) Update(
	context.Context,
	resource.UpdateRequest,
	*resource.UpdateResponse,
) {
}

// Read implements resource.Resource.
func (p *PackageResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data PackageResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	name := data.Name.ValueString()
	pkg, err := p.client.PackageGet(ctx, name)
	if err != nil {
		resp.State.RemoveResource(ctx)
	}

	pkgInfo, err := p.client.PackageFind(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to find package", err.Error())
		return
	}

	if data.Beta.IsNull() || data.Beta.IsUnknown() {
		resp.State.SetAttribute(ctx, path.Root("beta"), false)
	}

	if data.Version.IsNull() || data.Version.IsUnknown() {
		var version string
		if pkg != nil && pkg.Version != "" {
			version = pkg.Version
		} else if pkgInfo.Version != "" {
			version = pkgInfo.Version
		}
		data.Version = types.StringValue(version)
		resp.State.SetAttribute(ctx, path.Root("version"), version)
	}

	if data.URL.IsNull() || data.URL.IsUnknown() {
		if pkgInfo.Link != "" {
			data.URL = types.StringValue(pkgInfo.Link)
			resp.State.SetAttribute(ctx, path.Root("url"), pkgInfo.Link)
		}
	}

	// resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements resource.Resource.
func (p *PackageResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data PackageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	packageName := data.Name.ValueString()
	_, err := p.client.PackageUninstall(ctx, core.PackageUninstallRequest{
		ID: packageName,
	})
	if err != nil {
		_, err := p.client.PackageGet(ctx, packageName)
		switch err.(type) {
		case api.NotFoundError:
			resp.State.RemoveResource(ctx)
			return
		default:
			resp.Diagnostics.AddError("Failed to uninstall package", err.Error())
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

// Metadata implements resource.Resource.
func (p *PackageResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "package")
}

// Schema implements resource.Resource.
func (p *PackageResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Generic API Resource for making calls to the Synology DSM API.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the package to install.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The package version.",
				Optional:            true,
				Computed:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL to the package to install.",
				Optional:            true,
				Computed:            true,
			},
			"wizard": schema.MapAttribute{
				MarkdownDescription: "Wizard configuration values.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"file": schema.StringAttribute{
				MarkdownDescription: "The file to install.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"beta": schema.BoolAttribute{
				MarkdownDescription: "Whether to install beta versions of the package.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"run": schema.BoolAttribute{
				MarkdownDescription: "Whether to run the package after installation.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

func (f *PackageResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(synology.Api)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected client.Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)

		return
	}

	f.client = client.CoreAPI()
}

// ImportState implements resource.ResourceWithImportState.
func (p *PackageResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	pkg, err := p.client.PackageGet(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to find package", err.Error())
		return
	}

	pkgInfo, err := p.client.PackageFind(ctx, pkg.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to find package", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), pkg.ID)...)
	if pkg.Version != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version"), pkg.Version)...)
	}
	if pkgInfo.Link != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("url"), pkgInfo.Link)...)
	}
}
