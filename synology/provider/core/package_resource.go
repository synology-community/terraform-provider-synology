package core

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api"
	"github.com/synology-community/go-synology/pkg/api/core"
)

type PackageResourceModel struct {
	Name       types.String `tfsdk:"name"`
	Version    types.String `tfsdk:"version"`
	File       types.String `tfsdk:"file"`
	URL        types.String `tfsdk:"url"`
	Wizard     types.Map    `tfsdk:"wizard"`
	Beta       types.Bool   `tfsdk:"beta"`
	VolumePath types.String `tfsdk:"volume_path"`

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
		Name: data.Name.ValueString(),
		URL:  data.URL.ValueString(),
		Size: size,
		// Empty when unset, which is what asks the client to resolve a volume.
		VolumePath:  data.VolumePath.ValueString(),
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
//
// Only `run` is updatable. Every other attribute carries RequiresReplace, so
// the framework destroys and recreates rather than routing here -- DSM has no
// in-place upgrade or rename through this API, and pretending otherwise is how
// this function came to be an empty body that reported success while changing
// nothing on the NAS.
func (p *PackageResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state PackageResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()

	// Guard rather than assume: if anything other than `run` differs here, a
	// RequiresReplace modifier is missing and this would silently skip the
	// change -- the exact defect this rewrite exists to remove.
	if !plan.Name.Equal(state.Name) ||
		!plan.Version.Equal(state.Version) ||
		!plan.URL.Equal(state.URL) ||
		!plan.File.Equal(state.File) ||
		!plan.Beta.Equal(state.Beta) ||
		!plan.Wizard.Equal(state.Wizard) {
		resp.Diagnostics.AddError(
			"Unsupported in-place package update",
			fmt.Sprintf(
				"Package %q: only `run` can be changed in place; every other attribute "+
					"requires replacement. Reaching this point means a RequiresReplace plan "+
					"modifier is missing from the schema. Refusing rather than reporting a "+
					"success that would not reach the NAS.",
				name,
			),
		)
		return
	}

	if !plan.Run.Equal(state.Run) {
		var err error
		if plan.Run.ValueBool() {
			_, err = p.client.PackageStart(ctx, core.PackageControlRequest{ID: name})
		} else {
			_, err = p.client.PackageStop(ctx, core.PackageControlRequest{ID: name})
		}
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to set run state for package %q", name),
				err.Error(),
			)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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

	// Set unconditionally, not only when null or unknown. Backfilling only the
	// empty case meant a Package Center auto-update left state asserting the old
	// version forever, and `tofu plan` stayed clean while the NAS had moved on.
	var version string
	if pkg != nil && pkg.Version != "" {
		version = pkg.Version
	} else if pkgInfo.Version != "" {
		version = pkgInfo.Version
	}
	data.Version = types.StringValue(version)

	if data.URL.IsNull() || data.URL.IsUnknown() {
		if pkgInfo.Link != "" {
			data.URL = types.StringValue(pkgInfo.Link)
			resp.State.SetAttribute(ctx, path.Root("url"), pkgInfo.Link)
		}
	}

	// The single state write for this Read. It was commented out, so every
	// value computed above was discarded and no drift could ever surface.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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
		MarkdownDescription: `Manages packages on a Synology NAS using the Package Center API.

Install, configure, and manage Synology packages with optional wizard configuration for initial setup.

## Example Usage

` + "```hcl" + `
# Install MariaDB with wizard configuration
resource "synology_core_package" "mariadb" {
  name = "MariaDB10"
  
  wizard = {
    port              = "3306"
    new_root_password = "secure-password"
  }
}

# Install Docker package
resource "synology_core_package" "docker" {
  name = "Docker"
}
` + "```" + `
`,

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The Package Center identifier of the package to install, " +
					"for example `MariaDB10` or `ContainerManager`. DSM has no rename operation, " +
					"so changing this uninstalls the current package and installs a different one. " +
					"Uninstalling a package can remove the data it owns.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The package version. Changing it reinstalls the package, " +
					"because DSM offers no in-place upgrade through this API.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL to the package to install.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"wizard": schema.MapAttribute{
				MarkdownDescription: "Wizard configuration values, consumed only during installation.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"file": schema.StringAttribute{
				MarkdownDescription: "The file to install.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"beta": schema.BoolAttribute{
				MarkdownDescription: "Whether to install beta versions of the package.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"volume_path": schema.StringAttribute{
				MarkdownDescription: "The volume to install the package onto, for example " +
					"`/volume1`. Optional. When omitted the volume is resolved from DSM's " +
					"package settings: its configured default if it has one, otherwise the " +
					"NAS's only volume. On a multi-volume NAS with no configured default, " +
					"leaving this unset is an error rather than an arbitrary choice — which " +
					"volume a package lands on decides where its data lives, and DSM offers " +
					"no way to move it afterwards. Changing this reinstalls the package for " +
					"the same reason.",
				Optional: true,
				// Deliberately not Computed. The resolved volume is not reported
				// back by the package list, so a Computed attribute would leave
				// the framework unable to reconcile state after apply. Null here
				// honestly means "not specified", and the client resolves it.
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"run": schema.BoolAttribute{
				MarkdownDescription: "Whether the package should be running. Toggling this starts " +
					"or stops the package in place via `SYNO.Core.Package.Control`; it does not " +
					"reinstall it. This is the only attribute on this resource that can be updated " +
					"without replacement.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
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
