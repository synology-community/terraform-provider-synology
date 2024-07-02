package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/tredoe/osutil/user/crypt"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
)

// Ensure the implementation satisfies the desired interfaces.
var _ function.Function = &MkPasswdFunction{}

type MkPasswdFunction struct{}

func NewMkPasswdFunction() function.Function {
	return &MkPasswdFunction{}
}

func (f *MkPasswdFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "mkpasswd"
}

func (f *MkPasswdFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Creates a sha512 password hash that can be used with cloudinit.",
		Description: "Given a string password, returns the sha512 password hash.",

		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "password",
				Description: "The password to hash.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *MkPasswdFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string

	// Read Terraform argument data into the variable
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))

	if resp.Error != nil {
		return
	}

	c := crypt.New(crypt.SHA512)
	s := sha512_crypt.GetSalt()
	salt := s.Generate(s.SaltLenMax)

	result, err := c.Generate([]byte(input), salt)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
	}

	// Set the result to the same data
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
