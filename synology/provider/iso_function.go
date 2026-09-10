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

	iso, err := buildISOFromFiles(ctx, volumeName, files)
	if err != nil {
		resp.Error = function.NewFuncError(fmt.Sprintf("failed to create ISO: %v", err))
		return
	}

	resp.Error = function.ConcatFuncErrors(
		resp.Result.Set(ctx, iso),
	)
}

// buildISOFromFiles is the NAS-free body of ISOFunction.Run. Extracted so
// ordinary `go test` covers it; the TestAccISOFunction_* cases still need
// Terraform 1.8+ and a configured provider.
func buildISOFromFiles(
	ctx context.Context,
	volumeName string,
	files map[string]string,
) (string, error) {
	iso, err := util.IsoFromFiles(ctx, volumeName, files)
	if err != nil {
		return "", err
	}

	// Zero out the PVD volume descriptor timestamps to ensure deterministic output.
	// Terraform provider functions must be pure (same inputs -> same outputs), but the
	// iso9660 library embeds time.Now() in the Primary Volume Descriptor. The timestamps
	// are at offsets 813-880 within sector 16 (byte offset 32768).
	const pvdOffset = 16 * 2048 // sector 16
	if len(iso) >= pvdOffset+881 {
		// Zero VolumeCreation, VolumeModification, VolumeExpiration, VolumeEffective
		// timestamps (4 x 17 bytes at PVD offsets 813-880)
		for i := pvdOffset + 813; i <= pvdOffset+880; i++ {
			iso[i] = 0
		}
	}

	return string(iso), nil
}

func NewISOFunction() function.Function {
	return &ISOFunction{}
}
