package deno

import (
	"context"
	"fmt"
	"strings"

	"github.com/brad-jones/terraform-provider-denobridge/internal/jsocket"
	"github.com/hashicorp/terraform-plugin-framework/action"
)

// DenoClientAction is a client for executing Terraform actions using a Deno runtime.
// It wraps a DenoClient and provides action-specific functionality for invoking
// actions and receiving progress updates.
type DenoClientAction struct {
	// Client is the underlying Deno client used for JSON-RPC communication
	Client *DenoClient
}

// NewDenoClientAction creates a new DenoClientAction with the specified configuration.
// It initializes a Deno runtime process with the given script, permissions, and response handler.
//
// Parameters:
//   - denoBinaryPath: The path to the Deno executable
//   - scriptPath: The path to the TypeScript/JavaScript action script to execute
//   - configPath: The path to the Deno configuration file (deno.json)
//   - permissions: The Deno security permissions to grant the runtime
//   - resp: The Terraform action InvokeResponse for sending progress updates
//
// Returns a configured DenoClientAction ready to invoke actions.
func NewDenoClientAction(denoBinaryPath, scriptPath, configPath string, permissions *Permissions, resp *action.InvokeResponse) *DenoClientAction {
	return &DenoClientAction{
		NewDenoClient(
			denoBinaryPath,
			scriptPath,
			configPath,
			permissions,
			jsocket.TypedServerMethods(&DenoClientActionServerMethods{resp}),
		),
	}
}

// InvokeRequest represents the request payload for invoking a Terraform action.
// It contains the properties/parameters passed to the action from the Terraform configuration.
type InvokeRequest struct {
	// Props contains the action properties as defined in the Terraform schema
	Props any `json:"props"`
}

// InvokeResponse represents the response from invoking a Terraform action.
// It indicates whether the action has completed successfully.
type InvokeResponse struct {
	// Done indicates whether the action invocation completed successfully
	Done bool `json:"done"`
	// Diagnostics contains any warnings or errors to display to the user
	Diagnostics *[]struct {
		// Severity indicates the diagnostic level ("error" or "warning")
		Severity string `json:"severity"`
		// Summary is a short description of the diagnostic
		Summary string `json:"summary"`
		// Detail provides additional context about the diagnostic
		Detail string `json:"detail"`
		// PropPath optionally specifies which property the diagnostic relates to
		PropPath *[]string `json:"propPath,omitempty"`
	} `json:"diagnostics,omitempty"`
}

// Invoke executes the Terraform action by calling the "invoke" method via JSON-RPC.
// It sends the action properties to the Deno runtime and waits for completion.
//
// Parameters:
//   - ctx: The context for the operation, used for cancellation and timeouts
//   - params: The invoke request containing the action properties
//
// Returns an error if the JSON-RPC call fails or the action does not complete successfully.
func (c *DenoClientAction) Invoke(ctx context.Context, params *InvokeRequest) (*InvokeResponse, error) {
	var response *InvokeResponse
	if err := c.Client.Socket.Call(ctx, "invoke", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call invoke method over JSON-RPC: %v", err)
	}
	return response, nil
}

// DenoClientActionServerMethods implements the server-side JSON-RPC methods that
// the Deno runtime can call back to the provider. It handles progress updates
// during action execution.
type DenoClientActionServerMethods struct {
	// resp is the Terraform action response used to send progress updates
	resp *action.InvokeResponse
}

// InvokeProgressRequest represents a progress update request from the Deno runtime.
// It is sent during action execution to provide status updates to the user.
type InvokeProgressRequest struct {
	// Message is the progress message to display to the user
	Message string `json:"message"`
}

// InvokeProgress handles progress update requests from the Deno runtime during action execution.
// It forwards the progress message to Terraform for display to the user.
//
// Parameters:
//   - ctx: The context for the operation (currently unused but required by JSON-RPC interface)
//   - params: The progress request containing the message to display
func (c *DenoClientActionServerMethods) InvokeProgress(ctx context.Context, params *InvokeProgressRequest) {
	message := params.Message

	// ensure that the terraform cli output doesn't become misaligned.
	if !strings.HasSuffix(message, "\r") {
		message += "\r"
	}

	c.resp.SendProgress(action.InvokeProgressEvent{
		Message: message,
	})
}
