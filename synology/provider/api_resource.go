package provider

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api"
	"github.com/synology-community/terraform-provider-synology/synology/util"
)

type ApiResourceModel struct {
	API        types.String  `tfsdk:"api"`
	Method     types.String  `tfsdk:"method"`
	Version    types.Int64   `tfsdk:"version"`
	Parameters types.Map     `tfsdk:"parameters"`
	When       types.String  `tfsdk:"when"`
	Result     types.Dynamic `tfsdk:"result"`
}

var _ resource.Resource = &ApiResource{}

func NewApiResource() resource.Resource {
	return &ApiResource{}
}

type ApiResource struct {
	client synology.Api
}

func getParams(params map[string]string) any {
	sf := []reflect.StructField{}
	for k, v := range params {
		sf = append(sf, reflect.StructField{
			Name: strings.ToTitle(k),
			Type: reflect.TypeOf(v),
			Tag:  reflect.StructTag(fmt.Sprintf(`url:"%s"`, k)),
		})
	}

	typ := reflect.StructOf(sf)
	svl := reflect.New(typ).Elem()

	for k, v := range params {
		val := svl.FieldByName(strings.ToTitle(k))
		val.Set(reflect.ValueOf(v))
	}
	vt := svl.Interface()

	return vt
}

// Create implements resource.Resource.
func (a *ApiResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data ApiResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.When.ValueString() == "destroy" {
		return
	}

	params := map[string]string{}
	resp.Diagnostics.Append(data.Parameters.ElementsAs(ctx, &params, true)...)

	if resp.Diagnostics.HasError() {
		return
	}

	parameters := getParams(params)

	method := api.Method{
		API:            data.API.ValueString(),
		Method:         data.Method.ValueString(),
		Version:        int(data.Version.ValueInt64()),
		ErrorSummaries: api.GlobalErrors,
	}

	result, err := api.GetQuery[map[string]any](a.client, ctx, parameters, method)
	if err != nil {
		resp.Diagnostics.AddError("Failed to invoke API", err.Error())
		return
	}

	objValue, err := util.GetValue(*result)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get value", err.Error())
		return
	}

	data.Result = types.DynamicValue(objValue)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements resource.Resource.
func (a *ApiResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data ApiResourceModel
	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.When.ValueString() == "destroy" {

		params := map[string]string{}

		resp.Diagnostics.Append(data.Parameters.ElementsAs(ctx, &params, true)...)

		if resp.Diagnostics.HasError() {
			return
		}

		result, err := api.GetQuery[map[string]any](a.client, ctx, getParams(params), api.Method{
			API:            data.API.ValueString(),
			Method:         data.Method.ValueString(),
			Version:        int(data.Version.ValueInt64()),
			ErrorSummaries: api.GlobalErrors,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to invoke API", err.Error())
			return
		}

		objValue, err := util.GetValue(*result)
		if err != nil {
			resp.Diagnostics.AddError("Failed to get value", err.Error())
			return
		}

		data.Result = types.DynamicValue(objValue)

		// Save data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

// Metadata implements resource.Resource.
func (a *ApiResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_api"
}

// Read implements resource.Resource.
func (a *ApiResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {
}

// Schema implements resource.Resource.
func (a *ApiResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Generic API resource for making direct calls to any Synology DSM API.

Use this resource for API endpoints that don't have dedicated resources. Supports both apply-time and destroy-time operations.

## Example Usage

` + "```hcl" + `
# Get system information
resource "synology_api" "system_info" {
  api     = "SYNO.Core.System"
  method  = "info"
  version = 1
  
  parameters = {
    query = "all"
  }
}

output "system_info" {
  value = synology_api.system_info.result
}

# Run cleanup on destroy
resource "synology_api" "cleanup" {
  api     = "SYNO.Custom.Cleanup"
  method  = "run"
  version = 1
  when    = "destroy"
  
  parameters = {
    path = "/volume1/temp"
  }
}
` + "```" + `

See [examples/resources/synology_api](https://github.com/synology-community/terraform-provider-synology/tree/main/examples/resources/synology_api) for more examples.
`,

		Attributes: map[string]schema.Attribute{
			"api": schema.StringAttribute{
				MarkdownDescription: "The API to invoke.",
				Required:            true,
			},
			"method": schema.StringAttribute{
				MarkdownDescription: "The method to invoke.",
				Required:            true,
			},
			"version": schema.Int64Attribute{
				MarkdownDescription: "The version of the API to invoke.",
				Optional:            true,
			},
			"parameters": schema.MapAttribute{
				MarkdownDescription: "Name of the storage device.",
				Optional:            true,
				ElementType:         basetypes.StringType{},
			},
			"when": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("apply"),
				Validators: []validator.String{
					stringvalidator.OneOf("apply", "destroy"),
				},
			},
			"result": schema.DynamicAttribute{
				MarkdownDescription: "The result of the API call.",
				Computed:            true,
			},
		},
	}
}

// Update implements resource.Resource.
func (a *ApiResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan ApiResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.When.ValueString() == "destroy" {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	params := map[string]string{}
	resp.Diagnostics.Append(plan.Parameters.ElementsAs(ctx, &params, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parameters := getParams(params)

	method := api.Method{
		API:            plan.API.ValueString(),
		Method:         plan.Method.ValueString(),
		Version:        int(plan.Version.ValueInt64()),
		ErrorSummaries: api.GlobalErrors,
	}

	result, err := api.GetQuery[map[string]any](a.client, ctx, parameters, method)
	if err != nil {
		resp.Diagnostics.AddError("Failed to invoke API", err.Error())
		return
	}

	objValue, err := util.GetValue(*result)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get value", err.Error())
		return
	}

	plan.Result = types.DynamicValue(objValue)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (f *ApiResource) Configure(
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

	f.client = client
}
