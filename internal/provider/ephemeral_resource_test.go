package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestEphemeralResource(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	t.Setenv("TF_LOG", "DEBUG")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		// Ephemeral resources are only available in 1.10 and later
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		Steps: []resource.TestStep{
			{
				Config: `
					ephemeral "denobridge_ephemeral_resource" "test" {
						path = "./ephemeral_resource_test.ts"
						props = {
							type = "v4"
						}
					}

					provider "echo" {
						data = ephemeral.denobridge_ephemeral_resource.test
					}

					resource "echo" "test" {}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("result").AtMapKey("uuid"),
						knownvalue.StringRegexp(regexp.MustCompile(
							`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
						)),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("sensitive_result").AtMapKey("secret"),
						knownvalue.StringExact("ephemeral-secret"),
					),
				},
			},
		},
	})
}

func TestEphemeralResourceWithZod(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	t.Setenv("TF_LOG", "DEBUG")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		// Ephemeral resources are only available in 1.10 and later
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		Steps: []resource.TestStep{
			{
				Config: `
					ephemeral "denobridge_ephemeral_resource" "test_zod" {
						path = "./ephemeral_resource_zod_test.ts"
						props = {
							type = "v4"
						}
					}

					provider "echo" {
						data = ephemeral.denobridge_ephemeral_resource.test_zod
					}

					resource "echo" "test" {}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("result").AtMapKey("uuid"),
						knownvalue.StringRegexp(regexp.MustCompile(
							`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
						)),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("sensitive_result").AtMapKey("secret"),
						knownvalue.StringExact("ephemeral-secret"),
					),
				},
			},
		},
	})
}
