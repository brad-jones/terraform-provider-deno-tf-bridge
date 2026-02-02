package provider

import (
	"context"
	"fmt"

	"github.com/brad-jones/terraform-provider-denobridge/internal/deno"
	"github.com/brad-jones/terraform-provider-denobridge/internal/dynamic"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ action.Action              = &denoBridgeAction{}
	_ action.ActionWithConfigure = &denoBridgeAction{}
)

// NewDenoBridgeAction is a helper function to simplify the provider implementation.
func NewDenoBridgeAction() action.Action {
	return &denoBridgeAction{}
}

// denoBridgeAction defines the action implementation.
type denoBridgeAction struct {
	providerConfig *ProviderConfig
}

// denoBridgeActionModel maps the action schema data.
type denoBridgeActionModel struct {
	Path        types.String        `tfsdk:"path"`
	Props       types.Dynamic       `tfsdk:"props"`
	ConfigFile  types.String        `tfsdk:"config_file"`
	Permissions *deno.PermissionsTF `tfsdk:"permissions"`
}

func (a *denoBridgeAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action"
}

func (a *denoBridgeAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Bridges the terraform-plugin-framework Action to a Deno HTTP Server.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Description: "Path to the Deno script to execute.",
				Required:    true,
			},
			"props": schema.DynamicAttribute{
				Description: "Input properties to pass to the Deno script.",
				Optional:    true,
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

func (a *denoBridgeAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	providerConfig, ok := req.ProviderData.(*ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	a.providerConfig = providerConfig
}

func (a *denoBridgeAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	// Read Terraform configuration data into the model
	var data denoBridgeActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start the Deno server
	c := deno.NewDenoClientAction(
		a.providerConfig.DenoBinaryPath,
		data.Path.ValueString(),
		data.ConfigFile.ValueString(),
		data.Permissions.MapToDenoPermissions(),
		resp,
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

	// Call the invoke JSON-RPC method
	if err := c.Invoke(ctx, &deno.InvokeRequest{Props: dynamic.FromDynamic(data.Props)}); err != nil {
		resp.Diagnostics.AddError("Failed to invoke action", err.Error())
		return
	}
}
