package provider

import (
	"context"
	"fmt"

	"github.com/brad-jones/terraform-provider-denobridge/internal/deno"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider                       = &DenoBridgeProvider{}
	_ provider.ProviderWithActions            = &DenoBridgeProvider{}
	_ provider.ProviderWithEphemeralResources = &DenoBridgeProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DenoBridgeProvider{
			version: version,
		}
	}
}

// DenoBridgeProvider is the provider implementation.
type DenoBridgeProvider struct {
	version string
}

// denoBridgeProviderModel maps the provider schema data.
type denoBridgeProviderModel struct {
	DenoBinaryPath types.String `tfsdk:"deno_binary_path"`
	DenoVersion    types.String `tfsdk:"deno_version"`
}

// ProviderConfig holds the resolved provider configuration.
type ProviderConfig struct {
	DenoBinaryPath string
}

// Metadata returns the provider type name.
func (p *DenoBridgeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "denobridge"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *DenoBridgeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Deno Bridge provider enables Terraform to manage resources using Deno scripts.",
		Attributes: map[string]schema.Attribute{
			"deno_binary_path": schema.StringAttribute{
				MarkdownDescription: "Custom path to deno binary. When set, skips automatic download.",
				Optional:            true,
			},
			"deno_version": schema.StringAttribute{
				MarkdownDescription: "Deno version to auto-download (e.g., 'v2.1.4', 'v2.0.0-rc.1'). Defaults to 'latest' which downloads the latest stable GA release.",
				Optional:            true,
			},
		},
	}
}

// Configure is used to parse the provider config and create provider level services.
func (p *DenoBridgeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config denoBridgeProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve the Deno binary path
	var denoBinaryPath string

	if !config.DenoBinaryPath.IsNull() {
		// Use custom path if provided
		denoBinaryPath = config.DenoBinaryPath.ValueString()
	} else {
		// Auto-download Deno
		downloader := deno.NewDenoDownloader()

		version := "latest"
		if !config.DenoVersion.IsNull() {
			version = config.DenoVersion.ValueString()
		}

		path, err := downloader.GetDenoBinary(ctx, version)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to get Deno binary",
				fmt.Sprintf("Could not download or locate Deno binary: %s", err.Error()),
			)
			return
		}

		denoBinaryPath = path
	}

	// Create provider config
	providerConfig := &ProviderConfig{
		DenoBinaryPath: denoBinaryPath,
	}

	// Make available to resources and data sources
	resp.DataSourceData = providerConfig
	resp.ResourceData = providerConfig
	resp.EphemeralResourceData = providerConfig
	resp.ActionData = providerConfig
}

// Actions defines the actions implemented in the provider.
func (p *DenoBridgeProvider) Actions(_ context.Context) []func() action.Action {
	return []func() action.Action{
		NewDenoBridgeAction,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *DenoBridgeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDenoBridgeDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *DenoBridgeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDenoBridgeResource,
	}
}

// EphemeralResources defines the ephemeral resources implemented in the provider.
func (p *DenoBridgeProvider) EphemeralResources(_ context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		NewDenoBridgeEphemeralResource,
	}
}
