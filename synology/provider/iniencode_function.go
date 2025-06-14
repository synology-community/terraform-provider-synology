package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the desired interfaces.
var _ function.Function = &IniEncodeFunction{}

type IniEncodeFunction struct{}

func NewIniEncodeFunction() function.Function {
	return &IniEncodeFunction{}
}

func (f *IniEncodeFunction) Metadata(
	ctx context.Context,
	req function.MetadataRequest,
	resp *function.MetadataResponse,
) {
	resp.Name = "iniencode"
}

func (f *IniEncodeFunction) Definition(
	ctx context.Context,
	req function.DefinitionRequest,
	resp *function.DefinitionResponse,
) {
	resp.Definition = function.Definition{
		Summary:             "Encode an object to INI format",
		MarkdownDescription: "Encode an object to INI format",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:                "input",
				MarkdownDescription: "The object representation of an INI file.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *IniEncodeFunction) Run(
	ctx context.Context,
	req function.RunRequest,
	resp *function.RunResponse,
) {
	var input types.Dynamic

	// Read Terraform argument data into the variable
	resp.Error = req.Arguments.Get(ctx, &input)
	if resp.Error != nil {
		return
	}

	uv := input.UnderlyingValue()
	s, diags := encode(uv)
	if diags.HasError() {
		resp.Error = function.FuncErrorFromDiags(ctx, diags)
		return
	}

	svalue := types.StringValue(s)
	resp.Error = resp.Result.Set(ctx, &svalue)
}
