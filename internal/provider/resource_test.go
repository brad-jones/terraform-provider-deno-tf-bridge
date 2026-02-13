package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestResource(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	t.Setenv("TF_LOG", "DEBUG")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create test
			{
				Config: `
					resource "denobridge_resource" "test" {
						path  = "./resource_test.ts"
						props = {
							path = "./test.txt"
							content = "Hello World"
						}
						permissions = {
							all = true
						}
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("./test.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("props").AtMapKey("path"),
						knownvalue.StringExact("./test.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("props").AtMapKey("content"),
						knownvalue.StringExact("Hello World"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("state").AtMapKey("mtime"),
						knownvalue.Int64Func(func(v int64) error {
							if v > 0 {
								return nil
							}
							return fmt.Errorf("mtime not set")
						}),
					),
				},
			},
			// Update in place test
			{
				Config: `
					resource "denobridge_resource" "test" {
						path  = "./resource_test.ts"
						props = {
							path = "./test.txt"
							content = "Good Bye"
						}
						permissions = {
							all = true
						}
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("./test.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("props").AtMapKey("path"),
						knownvalue.StringExact("./test.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("props").AtMapKey("content"),
						knownvalue.StringExact("Good Bye"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("state").AtMapKey("mtime"),
						knownvalue.Int64Func(func(v int64) error {
							if v > 0 {
								return nil
							}
							return fmt.Errorf("mtime not set")
						}),
					),
				},
			},
			// Replacement test
			{
				Config: `
					resource "denobridge_resource" "test" {
						path  = "./resource_test.ts"
						props = {
							path = "./test2.txt"
							content = "Hello Again"
						}
						permissions = {
							all = true
						}
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("./test2.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("props").AtMapKey("path"),
						knownvalue.StringExact("./test2.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("props").AtMapKey("content"),
						knownvalue.StringExact("Hello Again"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("state").AtMapKey("mtime"),
						knownvalue.Int64Func(func(v int64) error {
							if v > 0 {
								return nil
							}
							return fmt.Errorf("mtime not set")
						}),
					),
				},
			},
			// Import test
			{
				ResourceName: "denobridge_resource.test",
				ImportState:  true,
				ImportStateId: `{
					"id": "./test2.txt",
					"path": "./resource_test.ts",
					"permissions": {
						"all": true
					}
				}`,
				ImportStateVerify: true,
			},
		},
	})
}

func TestResourceWithZod(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	t.Setenv("TF_LOG", "DEBUG")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create test
			{
				Config: `
					resource "denobridge_resource" "test_zod" {
						path  = "./resource_zod_test.ts"
						props = {
							path = "./test_zod.txt"
							content = "Hello Zod World"
						}
						permissions = {
							all = true
						}
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("id"),
						knownvalue.StringExact("./test_zod.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("props").AtMapKey("path"),
						knownvalue.StringExact("./test_zod.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("props").AtMapKey("content"),
						knownvalue.StringExact("Hello Zod World"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("state").AtMapKey("mtime"),
						knownvalue.Int64Func(func(v int64) error {
							if v > 0 {
								return nil
							}
							return fmt.Errorf("mtime not set")
						}),
					),
				},
			},
			// Update in place test
			{
				Config: `
					resource "denobridge_resource" "test_zod" {
						path  = "./resource_zod_test.ts"
						props = {
							path = "./test_zod.txt"
							content = "Good Bye Zod"
						}
						permissions = {
							all = true
						}
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("id"),
						knownvalue.StringExact("./test_zod.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("props").AtMapKey("path"),
						knownvalue.StringExact("./test_zod.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("props").AtMapKey("content"),
						knownvalue.StringExact("Good Bye Zod"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("state").AtMapKey("mtime"),
						knownvalue.Int64Func(func(v int64) error {
							if v > 0 {
								return nil
							}
							return fmt.Errorf("mtime not set")
						}),
					),
				},
			},
			// Replacement test
			{
				Config: `
					resource "denobridge_resource" "test_zod" {
						path  = "./resource_zod_test.ts"
						props = {
							path = "./test_zod2.txt"
							content = "Hello Again Zod"
						}
						permissions = {
							all = true
						}
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("id"),
						knownvalue.StringExact("./test_zod2.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("props").AtMapKey("path"),
						knownvalue.StringExact("./test_zod2.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("props").AtMapKey("content"),
						knownvalue.StringExact("Hello Again Zod"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("state").AtMapKey("mtime"),
						knownvalue.Int64Func(func(v int64) error {
							if v > 0 {
								return nil
							}
							return fmt.Errorf("mtime not set")
						}),
					),
				},
			},
			// Import test
			{
				ResourceName: "denobridge_resource.test_zod",
				ImportState:  true,
				ImportStateId: `{
					"id": "./test_zod2.txt",
					"path": "./resource_zod_test.ts",
					"permissions": {
						"all": true
					}
				}`,
				ImportStateVerify: true,
			},
		},
	})
}

func TestStatelessResource(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	t.Setenv("TF_LOG", "DEBUG")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create test
			{
				Config: `
					resource "denobridge_resource" "test" {
						path  = "./resource_test_stateless.ts"
						props = {
							path = "./test.txt"
							content = "Hello World"
						}
						permissions = {
							all = true
						}
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("./test.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("props").AtMapKey("path"),
						knownvalue.StringExact("./test.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("props").AtMapKey("content"),
						knownvalue.StringExact("Hello World"),
					),
				},
			},
			// Update in place test
			{
				Config: `
					resource "denobridge_resource" "test" {
						path  = "./resource_test_stateless.ts"
						props = {
							path = "./test.txt"
							content = "Good Bye"
						}
						permissions = {
							all = true
						}
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("./test.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("props").AtMapKey("path"),
						knownvalue.StringExact("./test.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("props").AtMapKey("content"),
						knownvalue.StringExact("Good Bye"),
					),
				},
			},
			// Replacement test
			{
				Config: `
					resource "denobridge_resource" "test" {
						path  = "./resource_test_stateless.ts"
						props = {
							path = "./test2.txt"
							content = "Hello Again"
						}
						permissions = {
							all = true
						}
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("./test2.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("props").AtMapKey("path"),
						knownvalue.StringExact("./test2.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test",
						tfjsonpath.New("props").AtMapKey("content"),
						knownvalue.StringExact("Hello Again"),
					),
				},
			},
			// Import test
			{
				ResourceName: "denobridge_resource.test",
				ImportState:  true,
				ImportStateId: `{
					"id": "./test2.txt",
					"path": "./resource_test_stateless.ts",
					"permissions": {
						"all": true
					}
				}`,
				ImportStateVerify: true,
			},
		},
	})
}

func TestStatelessResourceWithZod(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	t.Setenv("TF_LOG", "DEBUG")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create test
			{
				Config: `
					resource "denobridge_resource" "test_zod" {
						path  = "./resource_zod_test_stateless.ts"
						props = {
							path = "./test_zod.txt"
							content = "Hello Zod World"
						}
						permissions = {
							all = true
						}
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("id"),
						knownvalue.StringExact("./test_zod.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("props").AtMapKey("path"),
						knownvalue.StringExact("./test_zod.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("props").AtMapKey("content"),
						knownvalue.StringExact("Hello Zod World"),
					),
				},
			},
			// Update in place test
			{
				Config: `
					resource "denobridge_resource" "test_zod" {
						path  = "./resource_zod_test_stateless.ts"
						props = {
							path = "./test_zod.txt"
							content = "Good Bye Zod"
						}
						permissions = {
							all = true
						}
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("id"),
						knownvalue.StringExact("./test_zod.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("props").AtMapKey("path"),
						knownvalue.StringExact("./test_zod.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("props").AtMapKey("content"),
						knownvalue.StringExact("Good Bye Zod"),
					),
				},
			},
			// Replacement test
			{
				Config: `
					resource "denobridge_resource" "test_zod" {
						path  = "./resource_zod_test_stateless.ts"
						props = {
							path = "./test_zod2.txt"
							content = "Hello Again Zod"
						}
						permissions = {
							all = true
						}
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("id"),
						knownvalue.StringExact("./test_zod2.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("props").AtMapKey("path"),
						knownvalue.StringExact("./test_zod2.txt"),
					),
					statecheck.ExpectKnownValue(
						"denobridge_resource.test_zod",
						tfjsonpath.New("props").AtMapKey("content"),
						knownvalue.StringExact("Hello Again Zod"),
					),
				},
			},
			// Import test
			{
				ResourceName: "denobridge_resource.test_zod",
				ImportState:  true,
				ImportStateId: `{
					"id": "./test_zod2.txt",
					"path": "./resource_zod_test_stateless.ts",
					"permissions": {
						"all": true
					}
				}`,
				ImportStateVerify: true,
			},
		},
	})
}
