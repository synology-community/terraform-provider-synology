package container_test

import (
	"fmt"
	"regexp"
	"testing"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

func TestAccContainerOperationAction_basic(t *testing.T) {
	tests := []struct {
		name      string
		operation string
	}{
		{
			name:      "start operation",
			operation: "start",
		},
		{
			name:      "stop operation",
			operation: "stop",
		},
		{
			name:      "restart operation",
			operation: "restart",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r.UnitTest(t, r.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
				Steps: []r.TestStep{
					{
						Config: testAccContainerOperationActionConfig(tt.operation),
						Check: r.ComposeTestCheckFunc(
							r.TestCheckResourceAttr(
								"terraform_data.test_trigger",
								"input",
								tt.operation,
							),
						),
					},
				},
			})
		})
	}
}

func TestAccContainerOperationAction_withTrigger(t *testing.T) {
	r.UnitTest(t, r.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config: testAccContainerOperationActionWithTrigger(),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr(
						"terraform_data.trigger",
						"input",
						"test-trigger-value",
					),
				),
			},
		},
	})
}

func TestAccContainerOperationAction_multipleActions(t *testing.T) {
	r.UnitTest(t, r.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config: testAccContainerOperationActionMultiple(),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr(
						"terraform_data.multi_trigger",
						"input",
						"v1.0.0",
					),
				),
			},
		},
	})
}

func TestAccContainerOperationAction_invalidOperation(t *testing.T) {
	r.UnitTest(t, r.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config:      testAccContainerOperationActionConfigInvalid(),
				ExpectError: regexp.MustCompile(".*one of.*"),
			},
		},
	})
}

func testAccContainerOperationActionConfig(operation string) string {
	return fmt.Sprintf(`
provider "synology" {}

action "synology_container_operation" "test" {
  config {
    name      = "tinkerbell"
    operation = %[1]q
  }
}

resource "terraform_data" "test_trigger" {
  input = %[1]q

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.synology_container_operation.test]
    }
  }
}
`, operation)
}

func testAccContainerOperationActionWithTrigger() string {
	return `
provider "synology" {}

action "synology_container_operation" "start" {
  config {
    name      = "tinkerbell"
    operation = "start"
  }
}

resource "terraform_data" "trigger" {
  input = "test-trigger-value"

  lifecycle {
    action_trigger {
      events  = [after_create, after_update]
      actions = [action.synology_container_operation.start]
    }
  }
}
`
}

func testAccContainerOperationActionMultiple() string {
	return `
provider "synology" {}

action "synology_container_operation" "stop" {
  config {
    name      = "tinkerbell"
    operation = "stop"
  }
}

action "synology_container_operation" "start" {
  config {
    name      = "tinkerbell"
    operation = "start"
  }
}

resource "terraform_data" "multi_trigger" {
  input = "v1.0.0"

  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [
        action.synology_container_operation.stop,
        action.synology_container_operation.start
      ]
    }
  }
}
`
}

func testAccContainerOperationActionConfigInvalid() string {
	return `
provider "synology" {}

action "synology_container_operation" "invalid" {
  config {
    name      = "tinkerbell"
    operation = "invalid-operation"
  }
}
`
}
