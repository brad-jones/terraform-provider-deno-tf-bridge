package provider

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"denobridge": providerserver.NewProtocol6WithError(New("test")()),

	// The echoprovider is used to arrange tests by echoing ephemeral data
	// into the Terraform state so we can test our ephemeral_resource
	"echo": echoprovider.NewProviderServer(),
}

func testAccPreCheck(t *testing.T) {
	// Set TF_ACC_TERRAFORM_PATH to avoid downloading Terraform in CI
	// This allows tests to use the terraform binary that's already installed
	if os.Getenv("TF_ACC_TERRAFORM_PATH") == "" {
		if tfPath, err := exec.LookPath("terraform"); err == nil {
			t.Setenv("TF_ACC_TERRAFORM_PATH", tfPath)
		}
	}
}

// testAccProviderConfig returns the provider configuration block for acceptance tests.
// It checks for an existing Deno binary in PATH and configures the provider to use it,
// avoiding the need to download Deno during tests.
func testAccProviderConfig() string {
	denoBinary, err := exec.LookPath("deno")
	if err != nil {
		// If deno is not in PATH, return default config without deno_binary_path
		// which will trigger auto-download
		return `provider "denobridge" {}`
	}

	// Escape the path for safe inclusion in HCL configuration
	escapedPath := strings.ReplaceAll(denoBinary, `\`, `\\`)
	escapedPath = strings.ReplaceAll(escapedPath, `"`, `\"`)

	return `provider "denobridge" {
  deno_binary_path = "` + escapedPath + `"
}`
}
