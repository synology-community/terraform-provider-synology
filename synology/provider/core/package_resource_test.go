package core_test

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

// These tests install and uninstall real packages on a real NAS. They use
// r.Test, which skips unless TF_ACC is set. r.UnitTest -- used here
// previously -- sets IsUnitTest, and that flag's only effect is to bypass that
// very check, so `go test ./...` attempted a package install against whatever
// host happened to be configured.

type PackageResource struct{}

func TestAccPackageResource_basic(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		// {
		// 	"package name is set",
		// 	`
		// 	resource "synology_core_package" "foo" {
		// 		name = "exFAT-Free"
		// 	}`,
		// },
		// {
		// 	"package name is set",
		// 	`
		// 	resource "synology_core_package" "foo" {
		// 		name = "transmission"
		// 	}`,
		// },
		{
			"mariadb",
			`
			resource "synology_core_package" "mariadb" {
				name = "MariaDB10"

				wizard = {
					pkgwizard_port              = 3306
					pkgwizard_new_root_password = "T3stP@ssw0rd"
				}
			}`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			r.Test(t, r.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
				Steps: []r.TestStep{
					{
						Config: tt.ResourceBlock,
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttrWith(
								"synology_core_package.mariadb",
								"version",
								func(attr string) error {
									if attr == "" {
										return fmt.Errorf("expected package version to be set")
									}
									return nil
								},
							),
						),
					},
				},
			})
		})
	}
}

func TestAccPackageResource_url(t *testing.T) {
	testCases := []struct {
		Name          string
		ResourceBlock string
	}{
		{
			"package url is set",
			`
			resource "synology_core_package" "noble" {
				name = "vault"
				url = "https://synology-community.github.io/spksrc/packages/vault_1.17.2_linux_amd64.spk"
			}`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			r.Test(t, r.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
				Steps: []r.TestStep{
					{
						Config: tt.ResourceBlock,
						// The expected value has to be the URL the config above
						// actually sets. It previously asserted an Ubuntu cloud
						// image URL that appears nowhere in this test, so the
						// check could only ever fail.
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttr(
								"synology_core_package.noble",
								"url",
								"https://synology-community.github.io/spksrc/packages/vault_1.17.2_linux_amd64.spk",
							),
						),
					},
				},
			})
		})
	}
}

// packageConfig renders the MariaDB10 test package with a given `run` value.
// The wizard block is only consumed at install time, but it has to stay in
// every step's config: dropping it would register as an attribute change and
// plan a replacement rather than the in-place update under test.
func packageConfig(run bool) string {
	return fmt.Sprintf(`
	resource "synology_core_package" "mariadb" {
		name = "MariaDB10"
		run  = %t

		wizard = {
			pkgwizard_port              = 3306
			pkgwizard_new_root_password = "T3stP@ssw0rd"
		}
	}`, run)
}

// TestAccPackageResource_runTogglesInPlace is the end-to-end half of the
// carve-out. The schema test proves `run` carries no RequiresReplace; only
// this one proves the resulting Update actually reaches DSM and that the
// package survives it.
//
// The plan check is the real assertion. If `run` ever regains a
// RequiresReplace -- or Update quietly stops handling it -- the plan becomes a
// destroy-then-create, which on this resource means uninstalling MariaDB and
// taking its databases with it.
func TestAccPackageResource_runTogglesInPlace(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config: packageConfig(true),
				Check: r.TestCheckResourceAttr(
					"synology_core_package.mariadb", "run", "true"),
			},
			{
				Config: packageConfig(false),
				ConfigPlanChecks: r.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(
							"synology_core_package.mariadb",
							plancheck.ResourceActionUpdate,
						),
					},
				},
				Check: r.TestCheckResourceAttr(
					"synology_core_package.mariadb", "run", "false"),
			},
		},
	})
}

// TestAccPackageResource_readIsFaithful guards the other half of the defect.
// Read previously left `version` unset on one path and never wrote its result
// back to state at all, so state kept whatever the last apply had put there.
//
// A PlanOnly step re-plans the config that was just applied and requires the
// plan to be empty. That fails if Read invents a value, drops one, or reports
// something the device does not hold -- which is what let a wrong version sit
// in state and still plan clean.
//
// There is deliberately no acceptance test asserting that a version change
// plans a replacement. ConfigPlanChecks has no plan-only hook -- PreApply,
// PostApplyPreRefresh and PostApplyPostRefresh are all apply-relative -- so
// asserting it here would mean applying the replacement, and this resource
// replaces by uninstalling. That contract is covered instead by
// TestPackageSchema_AttributesDSMCannotChangeInPlaceRequireReplacement, which
// reads the plan modifier off the schema and costs no data.
func TestAccPackageResource_readIsFaithful(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config: packageConfig(true),
			},
			{
				Config:   packageConfig(true),
				PlanOnly: true,
			},
		},
	})
}
