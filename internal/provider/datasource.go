package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &denoBridgeDataSource{}
	_ datasource.DataSourceWithConfigure = &denoBridgeDataSource{}
)

// NewDenoBridgeDataSource is a helper function to simplify the provider implementation.
func NewDenoBridgeDataSource() datasource.DataSource {
	return &denoBridgeDataSource{}
}

// denoBridgeDataSource is the data source implementation.
type denoBridgeDataSource struct {
	providerConfig *ProviderConfig
}

// denoBridgeDataSourceModel maps the data source schema data.
type denoBridgeDataSourceModel struct {
	Path        types.String       `tfsdk:"path"`
	Props       types.Dynamic      `tfsdk:"props"`
	Result      types.Dynamic      `tfsdk:"result"`
	Permissions *denoPermissionsTF `tfsdk:"permissions"`
}

// Metadata returns the data source type name.
func (d *denoBridgeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_datasource"
}

// Schema defines the schema for the data source.
func (d *denoBridgeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Executes a Deno script via HTTP to fetch data.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Description: "Path to the Deno script to execute.",
				Required:    true,
			},
			"props": schema.DynamicAttribute{
				Description: "Input properties to pass to the Deno script.",
				Optional:    true,
			},
			"result": schema.DynamicAttribute{
				Description: "Output data returned from the Deno script.",
				Computed:    true,
			},
			"permissions": schema.SingleNestedAttribute{
				Description: "Deno runtime permissions for the script.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"all": schema.BoolAttribute{
						Description: "Grant all permissions.",
						Optional:    true,
					},
					"allow": schema.ListAttribute{
						Description: "List of permissions to allow (e.g., 'read', 'write', 'net').",
						ElementType: types.StringType,
						Optional:    true,
					},
					"deny": schema.ListAttribute{
						Description: "List of permissions to deny.",
						ElementType: types.StringType,
						Optional:    true,
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *denoBridgeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	providerConfig, ok := req.ProviderData.(*ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ProviderConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.providerConfig = providerConfig
}

// Read refreshes the Terraform state with the latest data.
func (d *denoBridgeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Get current state
	var state denoBridgeDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start the Deno server
	client := NewDenoClient(d.providerConfig.DenoBinaryPath, state.Path.ValueString(), state.Permissions.mapToDenoPermissions())
	if err := client.Start(ctx); err != nil {
		resp.Diagnostics.AddError(
			"Failed to start Deno server",
			fmt.Sprintf("Could not start Deno HTTP server: %s", err.Error()),
		)
		return
	}
	defer func() {
		if err := client.Stop(); err != nil {
			resp.Diagnostics.AddWarning(
				"Failed to stop Deno server",
				fmt.Sprintf("Could not stop Deno HTTP server: %s", err.Error()),
			)
		}
	}()

	// Call the read endpoint
	var result any
	if err := client.C().
		Post("/read").
		SetBody(map[string]any{"props": fromDynamic(state.Props)}).
		Do(ctx).
		Into(&result); err != nil {
		resp.Diagnostics.AddError(
			"Failed to read data",
			fmt.Sprintf("Could not read data from Deno script: %s", err.Error()),
		)
		return
	}

	// Set state
	state.Result = toDynamic(result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
