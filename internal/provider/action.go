package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

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
	Path        types.String       `tfsdk:"path"`
	Props       types.Dynamic      `tfsdk:"props"`
	Permissions *denoPermissionsTF `tfsdk:"permissions"`
}

func (a *denoBridgeAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action"
}

func (a *denoBridgeAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a resource via a Deno script with full CRUD lifecycle.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Description: "Path to the Deno script to execute.",
				Required:    true,
			},
			"props": schema.DynamicAttribute{
				Description: "Input properties to pass to the Deno script.",
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
	client := NewDenoClient(a.providerConfig.DenoBinaryPath, data.Path.ValueString(), data.Permissions.mapToDenoPermissions())
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

	// Call /invoke endpoint with the props
	httpResp, err := client.C().R().
		SetContext(ctx).
		SetBody(map[string]any{"props": fromDynamic(data.Props)}).
		Post("/invoke")

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to invoke action",
			fmt.Sprintf("Could not call /invoke endpoint: %s", err.Error()),
		)
		return
	}

	if httpResp.StatusCode != 200 {
		resp.Diagnostics.AddError(
			"Action invocation failed",
			fmt.Sprintf("Server returned status %d: %s", httpResp.StatusCode, httpResp.String()),
		)
		return
	}

	// Stream the JSONL response and send progress events
	if err := streamJSONLProgress(httpResp.Body, resp); err != nil {
		resp.Diagnostics.AddError(
			"Failed to process streaming response",
			fmt.Sprintf("Error reading streaming response: %s", err.Error()),
		)
		return
	}
}

// streamJSONLProgress reads a streaming JSONL response and sends progress events.
func streamJSONLProgress(body io.Reader, resp *action.InvokeResponse) error {
	scanner := bufio.NewScanner(body)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse the JSON line
		var progressData struct {
			Message string `json:"message"`
		}

		if err := json.Unmarshal([]byte(line), &progressData); err != nil {
			return fmt.Errorf("failed to parse JSONL line: %w", err)
		}

		// Send the progress event
		resp.SendProgress(action.InvokeProgressEvent{
			Message: progressData.Message,
		})
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading response stream: %w", err)
	}

	return nil
}
