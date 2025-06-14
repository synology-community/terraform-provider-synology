package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/terraform-provider-synology/synology/util"
)

var _ function.Function = &ISOFunction{}

type ISOFunction struct{}

func (r *ISOFunction) Metadata(
	_ context.Context,
	req function.MetadataRequest,
	resp *function.MetadataResponse,
) {
	resp.Name = "iso"
}

func (r *ISOFunction) Definition(
	ctx context.Context,
	req function.DefinitionRequest,
	resp *function.DefinitionResponse,
) {
	resp.Definition = function.Definition{
		Summary:             "Creates an ISO file from user data.",
		MarkdownDescription: "This function creates an ISO file from user data.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "name",
				MarkdownDescription: "The name of the volume for the iso.",
			},
			function.MapParameter{
				ElementType:         types.StringType,
				Name:                "files",
				MarkdownDescription: "A map of target file paths and the file content to add to the ISO file.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (r *ISOFunction) Run(
	ctx context.Context,
	req function.RunRequest,
	resp *function.RunResponse,
) {
	var volumeName string
	var files map[string]string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &volumeName, &files))
	if resp.Error != nil {
		return
	}

	iso, err := util.IsoFromFiles(ctx, volumeName, files)
	if err != nil {
		resp.Error = function.NewFuncError(fmt.Sprintf("failed to create ISO: %v", err))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Result.Set(ctx, iso))
}

func NewISOFunction() function.Function {
	return &ISOFunction{}
}
