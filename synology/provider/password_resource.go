package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tredoe/osutil/user/crypt"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
)

type PasswordResourceModel struct {
	Password types.String `tfsdk:"password"`
	Result   types.String `tfsdk:"result"`
}

var _ resource.Resource = &PasswordResource{}

func NewPasswordResource() resource.Resource {
	return &PasswordResource{}
}

type PasswordResource struct{}

// Create implements resource.Resource.
func (a *PasswordResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data PasswordResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	c := crypt.New(crypt.SHA512)
	s := sha512_crypt.GetSalt()
	salt := s.Generate(s.SaltLenMax)

	result, err := c.Generate([]byte(data.Password.ValueString()), salt)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate password hash", err.Error())
	}

	data.Result = types.StringValue(result)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete implements resource.Resource.
func (a *PasswordResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data PasswordResourceModel
	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Metadata implements resource.Resource.
func (a *PasswordResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_password"
}

// Read implements resource.Resource.
func (a *PasswordResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {
}

// Schema implements resource.Resource.
func (a *PasswordResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Resource for creating a password hash.",

		Attributes: map[string]schema.Attribute{
			"password": schema.StringAttribute{
				MarkdownDescription: "Password data.",
				Required:            true,
			},
			"result": schema.StringAttribute{
				MarkdownDescription: "The result of the API call.",
				Computed:            true,
			},
		},
	}
}

// Update implements resource.Resource.
func (a *PasswordResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data PasswordResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c := crypt.New(crypt.SHA512)
	s := sha512_crypt.GetSalt()
	salt := s.Generate(s.SaltLenMax)

	result, err := c.Generate([]byte(data.Password.ValueString()), salt)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate password hash", err.Error())
	}

	data.Result = types.StringValue(result)
}

func (f *PasswordResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
}
