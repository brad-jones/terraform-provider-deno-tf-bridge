package provider

import (
	"context"
	"fmt"

	"github.com/brad-jones/terraform-provider-denobridge/internal/deno"
	"github.com/brad-jones/terraform-provider-denobridge/internal/dynamic"
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
	Path        types.String        `tfsdk:"path"`
	Props       types.Dynamic       `tfsdk:"props"`
	Result      types.Dynamic       `tfsdk:"result"`
	ConfigFile  types.String        `tfsdk:"config_file"`
	Permissions *deno.PermissionsTF `tfsdk:"permissions"`
}

// Metadata returns the data source type name.
func (d *denoBridgeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_datasource"
}

// Schema defines the schema for the data source.
func (d *denoBridgeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Bridges the terraform-plugin-framework Datasource to a Deno HTTP Server.",
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
			"config_file": schema.StringAttribute{
				Description: "File path to a deno config file to use with the deno script. Useful for import maps, etc...",
				Optional:    true,
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
	c := deno.NewDenoClientDatasource(
		d.providerConfig.DenoBinaryPath,
		state.Path.ValueString(),
		state.ConfigFile.ValueString(),
		state.Permissions.MapToDenoPermissions(),
	)
	if err := c.Client.Start(ctx); err != nil {
		resp.Diagnostics.AddError("Failed to start Deno", err.Error())
		return
	}
	defer func() {
		if err := c.Client.Stop(); err != nil {
			resp.Diagnostics.AddWarning("Failed to stop Deno", err.Error())
		}
	}()

	// Call the read JSON-RPC method
	response, err := c.Read(ctx, &deno.ReadRequest{Props: dynamic.FromDynamic(state.Props)})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read data",
			fmt.Sprintf("Could not read data from Deno script: %s", err.Error()),
		)
	}

	// Set state
	state.Result = dynamic.ToDynamic(response.Result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
