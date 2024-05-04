package provider_test

import (
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type GuestsDataSource struct{}

func TestAccGuestsDataSource_basic(t *testing.T) {
	testCases := []struct {
		Name            string
		DataSourceBlock string
		Expected        string
	}{
		{
			"no gzip or b64 - basic content",
			`data "synology_guests" "foo" {}`,
			"Content-Type: multipart/mixed; boundary=\"MIMEBOUNDARY\"\nMIME-Version: 1.0\r\n\r\n--MIMEBOUNDARY\r\nContent-Transfer-Encoding: 7bit\r\nContent-Type: text/x-shellscript\r\nMime-Version: 1.0\r\n\r\nbaz\r\n--MIMEBOUNDARY--\r\n",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			r.UnitTest(t, r.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []r.TestStep{
					{
						Config: tt.DataSourceBlock,
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttrSet("data.synology_guests.foo", "guest.0.%"),
						),
					},
				},
			})
		})
	}
}
