package provider

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/appkins/terraform-provider-synology/synology/util"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api"
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

// Create implements resource.Resource.
func (a *ApiResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ApiResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]string{}

	resp.Diagnostics.Append(data.Parameters.ElementsAs(ctx, &params, true)...)

	if resp.Diagnostics.HasError() {
		return
	}

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

	method := api.Method{
		API:          data.API.ValueString(),
		Method:       data.Method.ValueString(),
		Version:      int(data.Version.ValueInt64()),
		ErrorSummary: api.GlobalErrors,
	}

	result, err := api.GetQuery[map[string]any](a.client, ctx, vt, method)

	if err != nil {
		resp.Diagnostics.AddError("Failed to invoke API", err.Error())
		return
	}

	objValue, err := util.GetValue(result)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get value", err.Error())
		return
	}

	data.Result = types.DynamicValue(objValue)

	// attrValue, err := util.GetValue(&result)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get value", err.Error())
	// 	return
	// }
	// attrTypes, err := util.GetType(&result)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get type", err.Error())
	// 	return
	// }

	// objValue := types.ObjectValueMust(attrTypes, attrValue)

	// data.Result = types.DynamicValue(objValue)

	// resultValue := map[string]attr.Value{}

	// for key, value := range *result {
	// 	vT := reflect.TypeOf(value)
	// 	switch vT.Kind() {
	// 	case reflect.Map:
	// 		resultValue[key] = types.MapValueMust(
	// 			types.StringType,
	// 			vT.
	// 	}
	// 	rv, err := json.Marshal(value)
	// 	if err == nil {
	// 		resultValue[key] = types.StringValue(string(rv))
	// 	}
	// }

	// data.Result = types.MapValueMust(
	// 	types.StringType,
	// 	resultValue,
	// )

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

// Delete implements resource.Resource.
func (a *ApiResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {

}

// Metadata implements resource.Resource.
func (a *ApiResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api"
}

// Read implements resource.Resource.
func (a *ApiResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {

}

// Schema implements resource.Resource.
func (a *ApiResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Generic API Resource for making calls to the Synology DSM API.",

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
func (a *ApiResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {

}

func (f *ApiResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(synology.Api)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	f.client = client
}
