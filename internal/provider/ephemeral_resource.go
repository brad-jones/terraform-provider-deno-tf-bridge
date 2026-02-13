package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brad-jones/terraform-provider-denobridge/internal/deno"
	"github.com/brad-jones/terraform-provider-denobridge/internal/dynamic"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ ephemeral.EphemeralResource              = &denoBridgeEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &denoBridgeEphemeralResource{}
	_ ephemeral.EphemeralResourceWithRenew     = &denoBridgeEphemeralResource{}
	_ ephemeral.EphemeralResourceWithClose     = &denoBridgeEphemeralResource{}
)

// NewDenoBridgeEphemeralResource is a helper function to simplify the provider implementation.
func NewDenoBridgeEphemeralResource() ephemeral.EphemeralResource {
	return &denoBridgeEphemeralResource{}
}

// denoBridgeEphemeralResource is the resource implementation.
type denoBridgeEphemeralResource struct {
	providerConfig *ProviderConfig
}

// denoBridgeEphemeralResourceModel maps the resource schema data.
type denoBridgeEphemeralResourceModel struct {
	Path            types.String        `tfsdk:"path"`
	Props           types.Dynamic       `tfsdk:"props"`
	Result          types.Dynamic       `tfsdk:"result"`
	SensitiveResult types.Dynamic       `tfsdk:"sensitive_result"`
	ConfigFile      types.String        `tfsdk:"config_file"`
	Permissions     *deno.PermissionsTF `tfsdk:"permissions"`
}

func (r *denoBridgeEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ephemeral_resource"
}

func (r *denoBridgeEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Bridges the terraform-plugin-framework Ephemeral Resource to a Deno script.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Description: "Path to the Deno script to execute.",
				Required:    true,
			},
			"props": schema.DynamicAttribute{
				Description: "Input properties to pass to the Deno script.",
				Required:    true,
			},
			"result": schema.DynamicAttribute{
				Description: "Output data returned from the Deno script.",
				Computed:    true,
			},
			"sensitive_result": schema.DynamicAttribute{
				Description: "Sensitive output data returned from the Deno script.",
				Computed:    true,
				Sensitive:   true,
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
func (r *denoBridgeEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

	r.providerConfig = providerConfig
}

func (r *denoBridgeEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	// Read Terraform config data into the model
	var data denoBridgeEphemeralResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start the Deno server
	c := deno.NewDenoClientEphemeralResource(
		r.providerConfig.DenoBinaryPath,
		data.Path.ValueString(),
		data.ConfigFile.ValueString(),
		data.Permissions.MapToDenoPermissions(),
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

	// Call the open endpoint
	response, err := c.Open(ctx, &deno.OpenRequest{Props: dynamic.FromDynamic(data.Props)})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to open data",
			fmt.Sprintf("Could not open data from Deno script: %s", err.Error()),
		)
	}

	// Handle diagnostics - allows the script to add warnings or errors
	if response.Diagnostics != nil {
		fatal := false
		for _, diag := range *response.Diagnostics {
			switch diag.Severity {
			case "error":
				fatal = true
				if diag.PropPath != nil {
					resp.Diagnostics.AddAttributeError(dynamic.PropPathToPath(diag.PropPath), diag.Summary, diag.Detail)
				} else {
					resp.Diagnostics.AddError(diag.Summary, diag.Detail)
				}
			case "warning":
				if diag.PropPath != nil {
					resp.Diagnostics.AddAttributeWarning(dynamic.PropPathToPath(diag.PropPath), diag.Summary, diag.Detail)
				} else {
					resp.Diagnostics.AddWarning(diag.Summary, diag.Detail)
				}
			}
		}
		if fatal {
			return
		}
	}

	// Set a renew time if provided
	if response.RenewAt != nil {
		resp.RenewAt = time.Unix(*response.RenewAt, 0)
	}

	// Set any private data
	if response.Private != nil {
		privateJSON, err := json.Marshal(*response.Private)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to marshal private data",
				fmt.Sprintf("Could not marshal private data to JSON: %s", err.Error()),
			)
			return
		}
		resp.Private.SetKey(ctx, "data", privateJSON)
	}

	// Save config into a private key so we can easily get it in renew and close
	configJSON, err := json.Marshal(map[string]any{
		"DenoBinaryPath":  r.providerConfig.DenoBinaryPath,
		"DenoScriptPath":  data.Path.ValueString(),
		"DenoConfigPath":  data.ConfigFile.ValueString(),
		"DenoPermissions": data.Permissions.MapToDenoPermissions(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to marshal private config",
			fmt.Sprintf("Could not marshal private config to JSON: %s", err.Error()),
		)
		return
	}
	resp.Private.SetKey(ctx, "config", configJSON)

	// Set result
	data.Result = dynamic.ToDynamic(response.Result)
	data.SensitiveResult = dynamic.ToDynamic(response.SensitiveResult)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

func (r *denoBridgeEphemeralResource) Renew(ctx context.Context, req ephemeral.RenewRequest, resp *ephemeral.RenewResponse) {
	// Read config
	privateConfigBytes, diags := req.Private.GetKey(ctx, "config")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var privateConfig struct {
		DenoBinaryPath  string
		DenoScriptPath  string
		DenoConfigPath  string
		DenoPermissions *deno.Permissions
	}
	err := json.Unmarshal(privateConfigBytes, &privateConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to unmarshal private config",
			fmt.Sprintf("Could not unmarshal private config from JSON: %s", err.Error()),
		)
		return
	}

	// Read data
	privateDataBytes, diags := req.Private.GetKey(ctx, "data")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var privateData *any
	if len(privateDataBytes) > 0 {
		err = json.Unmarshal(privateDataBytes, &privateData)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to unmarshal private data",
				fmt.Sprintf("Could not unmarshal private data from JSON: %s", err.Error()),
			)
			return
		}
	}

	// Start the Deno server
	c := deno.NewDenoClientEphemeralResource(
		privateConfig.DenoBinaryPath,
		privateConfig.DenoScriptPath,
		privateConfig.DenoConfigPath,
		privateConfig.DenoPermissions,
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

	// Call the renew endpoint
	response, err := c.Renew(ctx, &deno.RenewRequest{Private: privateData})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to renew",
			fmt.Sprintf("Could not renew data from Deno script: %s", err.Error()),
		)
		return
	}

	// Handle diagnostics - allows the script to add warnings or errors
	if response.Diagnostics != nil {
		fatal := false
		for _, diag := range *response.Diagnostics {
			switch diag.Severity {
			case "error":
				fatal = true
				if diag.PropPath != nil {
					resp.Diagnostics.AddAttributeError(dynamic.PropPathToPath(diag.PropPath), diag.Summary, diag.Detail)
				} else {
					resp.Diagnostics.AddError(diag.Summary, diag.Detail)
				}
			case "warning":
				if diag.PropPath != nil {
					resp.Diagnostics.AddAttributeWarning(dynamic.PropPathToPath(diag.PropPath), diag.Summary, diag.Detail)
				} else {
					resp.Diagnostics.AddWarning(diag.Summary, diag.Detail)
				}
			}
		}
		if fatal {
			return
		}
	}

	// Set a new renew time if provided
	if response.RenewAt != nil {
		resp.RenewAt = time.Unix(*response.RenewAt, 0)
	}

	// Set new private data if provided
	if response.Private != nil {
		privateJSON, err := json.Marshal(*response.Private)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to marshal private data",
				fmt.Sprintf("Could not marshal private data to JSON: %s", err.Error()),
			)
			return
		}
		resp.Private.SetKey(ctx, "data", privateJSON)
	}
}

func (r *denoBridgeEphemeralResource) Close(ctx context.Context, req ephemeral.CloseRequest, resp *ephemeral.CloseResponse) {
	// Read config
	privateConfigBytes, diags := req.Private.GetKey(ctx, "config")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var privateConfig struct {
		DenoBinaryPath  string
		DenoScriptPath  string
		DenoConfigPath  string
		DenoPermissions *deno.Permissions
	}
	err := json.Unmarshal(privateConfigBytes, &privateConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to unmarshal private config",
			fmt.Sprintf("Could not unmarshal private config from JSON: %s", err.Error()),
		)
		return
	}

	// Read data
	privateDataBytes, diags := req.Private.GetKey(ctx, "data")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var privateData *any
	if len(privateDataBytes) > 0 {
		err = json.Unmarshal(privateDataBytes, &privateData)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to unmarshal private data",
				fmt.Sprintf("Could not unmarshal private data from JSON: %s", err.Error()),
			)
			return
		}
	}

	// Start the Deno server
	c := deno.NewDenoClientEphemeralResource(
		privateConfig.DenoBinaryPath,
		privateConfig.DenoScriptPath,
		privateConfig.DenoConfigPath,
		privateConfig.DenoPermissions,
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

	// Call the close endpoint
	response, err := c.Close(ctx, &deno.CloseRequest{Private: privateData})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to close",
			fmt.Sprintf("Could not close data from Deno script: %s", err.Error()),
		)
		return
	}

	// The close method is optional
	if response == nil {
		return
	}

	// Handle diagnostics - allows the script to add warnings or errors
	if response.Diagnostics != nil {
		fatal := false
		for _, diag := range *response.Diagnostics {
			switch diag.Severity {
			case "error":
				fatal = true
				if diag.PropPath != nil {
					resp.Diagnostics.AddAttributeError(dynamic.PropPathToPath(diag.PropPath), diag.Summary, diag.Detail)
				} else {
					resp.Diagnostics.AddError(diag.Summary, diag.Detail)
				}
			case "warning":
				if diag.PropPath != nil {
					resp.Diagnostics.AddAttributeWarning(dynamic.PropPathToPath(diag.PropPath), diag.Summary, diag.Detail)
				} else {
					resp.Diagnostics.AddWarning(diag.Summary, diag.Detail)
				}
			}
		}
		if fatal {
			return
		}
	}

	// Double check that the operation actually completed
	if !response.Done {
		resp.Diagnostics.AddError(
			"Failed to close resource",
			"Deno script did not report the operation as done",
		)
		return
	}
}
