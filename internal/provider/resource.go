package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brad-jones/terraform-provider-denobridge/internal/deno"
	"github.com/brad-jones/terraform-provider-denobridge/internal/dynamic"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &denoBridgeResource{}
	_ resource.ResourceWithConfigure   = &denoBridgeResource{}
	_ resource.ResourceWithModifyPlan  = &denoBridgeResource{}
	_ resource.ResourceWithImportState = &denoBridgeResource{}
)

// NewDenoBridgeResource is a helper function to simplify the provider implementation.
func NewDenoBridgeResource() resource.Resource {
	return &denoBridgeResource{}
}

// denoBridgeResource is the resource implementation.
type denoBridgeResource struct {
	providerConfig *ProviderConfig
}

// denoBridgeResourceModel maps the resource schema data.
type denoBridgeResourceModel struct {
	ID          types.String        `tfsdk:"id"`
	Path        types.String        `tfsdk:"path"`
	Props       types.Dynamic       `tfsdk:"props"`
	State       types.Dynamic       `tfsdk:"state"`
	ConfigFile  types.String        `tfsdk:"config_file"`
	Permissions *deno.PermissionsTF `tfsdk:"permissions"`
}

// Metadata returns the resource type name.
func (r *denoBridgeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

// Schema defines the schema for the resource.
func (r *denoBridgeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Bridges the terraform-plugin-framework Resource to a Deno script.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "Path to the Deno script to execute.",
				Required:    true,
			},
			"props": schema.DynamicAttribute{
				Description: "Input properties to pass to the Deno script.",
				Optional:    true,
			},
			"state": schema.DynamicAttribute{
				Description: "Additional computed state of the resource as returned by the Deno script.",
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

// Configure adds the provider configured client to the resource.
func (r *denoBridgeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.providerConfig = providerConfig
}

// Create creates the resource and sets the initial Terraform state.
func (r *denoBridgeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan denoBridgeResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start the Deno server
	c := deno.NewDenoClientResource(
		r.providerConfig.DenoBinaryPath,
		plan.Path.ValueString(),
		plan.ConfigFile.ValueString(),
		plan.Permissions.MapToDenoPermissions(),
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

	// Call the create endpoint
	response, err := c.Create(ctx, &deno.CreateRequest{Props: dynamic.FromDynamic(plan.Props)})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create resource",
			fmt.Sprintf("Could not create resource via Deno script: %s", err.Error()),
		)
		return
	}

	// Set state
	plan.ID = types.StringValue(response.ID)
	if response.State != nil {
		plan.State = dynamic.ToDynamic(response.State)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *denoBridgeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state denoBridgeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start the Deno server
	c := deno.NewDenoClientResource(
		r.providerConfig.DenoBinaryPath,
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

	// Call the read endpoint
	response, err := c.Read(ctx, &deno.CreateReadRequest{ID: state.ID.ValueString(), Props: dynamic.FromDynamic(state.Props)})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read resource",
			fmt.Sprintf("Could not read resource via Deno script: %s", err.Error()),
		)
		return
	}

	if response.Exists != nil && !*response.Exists {
		resp.State.RemoveResource(ctx)
		return
	}

	// Set refreshed state
	state.Props = dynamic.ToDynamic(response.Props)
	if response.State != nil {
		state.State = dynamic.ToDynamic(response.State)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *denoBridgeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan denoBridgeResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state for ID
	var state denoBridgeResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start the Deno server
	c := deno.NewDenoClientResource(
		r.providerConfig.DenoBinaryPath,
		plan.Path.ValueString(),
		plan.ConfigFile.ValueString(),
		plan.Permissions.MapToDenoPermissions(),
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

	// Call the update endpoint
	response, err := c.Update(ctx, &deno.UpdateRequest{
		ID:           state.ID.ValueString(),
		NextProps:    dynamic.FromDynamic(plan.Props),
		CurrentProps: dynamic.FromDynamic(state.Props),
		CurrentState: dynamic.FromDynamic(state.State),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update resource",
			fmt.Sprintf("Could not update resource via Deno script: %s", err.Error()),
		)
		return
	}

	// Keep the same ID
	plan.ID = state.ID

	// Set updated state
	if response.State != nil {
		plan.State = dynamic.ToDynamic(response.State)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *denoBridgeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state denoBridgeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start the Deno server
	c := deno.NewDenoClientResource(
		r.providerConfig.DenoBinaryPath,
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

	// Call the delete endpoint
	if err := c.Delete(ctx, &deno.DeleteRequest{
		ID:    state.ID.ValueString(),
		Props: dynamic.FromDynamic(state.Props),
		State: dynamic.FromDynamic(state.State),
	}); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete resource",
			fmt.Sprintf("Could not delete resource via Deno script: %s", err.Error()),
		)
		return
	}
}

// ModifyPlan calls the Deno script's optional /modify-plan endpoint to allow custom plan modification.
// The script can return modified props, specify attributes requiring replacement, and add diagnostics.
func (r *denoBridgeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Get the plan data, if it exists
	var plan *denoBridgeResourceModel
	if !req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Get the state data, if it exists
	var state *denoBridgeResourceModel
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Bail out early if nothing is actually changing for updates
	if plan != nil && state != nil {
		if plan.Props.Equal(state.Props) {
			return
		}
	}

	// Get the deno script from the plan for create & update operations.
	// Otherwise for delete we get the details from the existing state.
	var denoScriptPath string
	var denoConfigPath string
	var denoPermissions *deno.PermissionsTF
	if plan != nil {
		denoScriptPath = plan.Path.ValueString()
		denoConfigPath = plan.ConfigFile.ValueString()
		denoPermissions = plan.Permissions
	} else {
		if state != nil {
			denoScriptPath = state.Path.ValueString()
			denoConfigPath = state.ConfigFile.ValueString()
			denoPermissions = state.Permissions
		}
	}

	// Bail out if we can't call deno
	if denoScriptPath == "" || denoPermissions == nil {
		resp.Diagnostics.AddWarning("ModifyPlan SKIPPED", "missing denoScriptPath or denoPermissions")
		return
	}

	// Start the Deno server
	c := deno.NewDenoClientResource(
		r.providerConfig.DenoBinaryPath,
		denoScriptPath,
		denoConfigPath,
		denoPermissions.MapToDenoPermissions(),
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

	// Build the request payload
	var id *string
	if state != nil {
		id = state.ID.ValueStringPointer()
	}
	planType := ""
	var nextProps any
	var currentProps any
	var currentState any
	if plan != nil && state == nil {
		planType = "create"
		nextProps = dynamic.FromDynamic(plan.Props)
	}
	if plan != nil && state != nil {
		planType = "update"
		nextProps = dynamic.FromDynamic(plan.Props)
		currentProps = dynamic.FromDynamic(state.Props)
		currentState = dynamic.FromDynamic(state.State)
	}
	if plan == nil && state != nil {
		planType = "delete"
		currentProps = dynamic.FromDynamic(state.Props)
		currentState = dynamic.FromDynamic(state.State)
	}

	response, err := c.ModifyPlan(ctx, &deno.ModifyPlanRequest{
		ID:           id,
		PlanType:     planType,
		NextProps:    nextProps,
		CurrentProps: currentProps,
		CurrentState: currentState,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to modify the plan", err.Error())
		return
	}

	// Bail out if there is nothing to modify
	if response == nil || response.NoChanges != nil && *response.NoChanges {
		return
	}

	// Handle modified props - allows the script to modify the planned properties
	if response.ModifiedProps != nil {
		plan.Props = dynamic.ToDynamic(response.ModifiedProps)
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
	}

	// Handle requiresReplacement - instructing tf to do a create then delete instead of an update
	if response.RequiresReplacement != nil && *response.RequiresReplacement {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("props"))
	}

	// Handle diagnostics - allows the script to add warnings or errors
	// Mainly for use with the Resource Destroy Plan Diagnostics workflow.
	// see: https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification#resource-destroy-plan-diagnostics
	if response.Diagnostics != nil {
		for _, diag := range *response.Diagnostics {
			switch diag.Severity {
			case "error":
				if diag.PropName != nil {
					resp.Diagnostics.AddAttributeError(path.Root("props").AtMapKey(*diag.PropName), diag.Summary, diag.Detail)
				} else {
					resp.Diagnostics.AddError(diag.Summary, diag.Detail)
				}
			case "warning":
				if diag.PropName != nil {
					resp.Diagnostics.AddAttributeWarning(path.Root("props").AtMapKey(*diag.PropName), diag.Summary, diag.Detail)
				} else {
					resp.Diagnostics.AddWarning(diag.Summary, diag.Detail)
				}
			}
		}
	}
}

// ImportState imports an existing resource into Terraform state.
// The import ID must be a JSON string containing the resource ID, Deno script path,
// and any required permissions. Props are optional and should only include properties
// needed to uniquely identify the resource (resource-dependent).
func (r *denoBridgeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var importConfig struct {
		ID          string            `json:"id"`
		Path        string            `json:"path"`
		Props       *map[string]any   `json:"props,omitempty"`
		ConfigFile  *string           `json:"config_file,omitempty"`
		Permissions *deno.Permissions `json:"permissions,omitempty"`
	}
	err := json.Unmarshal([]byte(req.ID), &importConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID Format",
			fmt.Sprintf("Import ID must be valid JSON containing id, path, and optional props/permissions. Error: %s", err.Error()),
		)
		return
	}

	var props types.Dynamic
	if importConfig.Props != nil {
		props = dynamic.ToDynamic(importConfig.Props)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, denoBridgeResourceModel{
		ID:          types.StringValue(importConfig.ID),
		Path:        types.StringValue(importConfig.Path),
		Props:       props,
		ConfigFile:  types.StringPointerValue(importConfig.ConfigFile),
		Permissions: importConfig.Permissions.MapToDenoPermissionsTF(),
	})...)
}
