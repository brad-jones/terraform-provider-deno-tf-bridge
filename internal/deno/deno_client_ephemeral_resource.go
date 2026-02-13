package deno

import (
	"context"
	"errors"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
)

// DenoClientEphemeralResource is a client for managing Terraform ephemeral resources using a Deno runtime.
// It wraps a DenoClient and provides ephemeral resource-specific functionality for opening,
// renewing, and closing short-lived resources within a Terraform plan or apply operation.
type DenoClientEphemeralResource struct {
	// Client is the underlying Deno client used for JSON-RPC communication
	Client *DenoClient
}

// NewDenoClientEphemeralResource creates a new DenoClientEphemeralResource with the specified configuration.
// It initializes a Deno runtime process with the given script and permissions.
//
// Parameters:
//   - denoBinaryPath: The path to the Deno executable
//   - scriptPath: The path to the TypeScript/JavaScript ephemeral resource script to execute
//   - configPath: The path to the Deno configuration file (deno.json)
//   - permissions: The Deno security permissions to grant the runtime
//
// Returns a configured DenoClientEphemeralResource ready to manage ephemeral resources.
func NewDenoClientEphemeralResource(denoBinaryPath, scriptPath, configPath string, permissions *Permissions) *DenoClientEphemeralResource {
	return &DenoClientEphemeralResource{
		NewDenoClient(
			denoBinaryPath,
			scriptPath,
			configPath,
			permissions,
			nil,
		),
	}
}

// OpenRequest represents the request payload for opening an ephemeral resource.
// It contains the configuration properties passed from the Terraform configuration.
type OpenRequest struct {
	// Props contains the ephemeral resource configuration properties as defined in the Terraform schema
	Props any `json:"props"`
}

// OpenResponse represents the response from opening an ephemeral resource.
// It contains the resource data, optional renewal time, and private state data.
type OpenResponse struct {
	// Result contains the ephemeral resource data to be made available during the Terraform operation
	Result any `json:"result"`
	// SensitiveResult contains the ephemeral resource sensitive data (marked as sensitive in Terraform)
	SensitiveResult any `json:"sensitiveResult"`
	// RenewAt is an optional Unix timestamp (in seconds) indicating when the resource should be renewed
	RenewAt *int64 `json:"renewAt,omitempty"`
	// Private is optional private state data that will be passed to subsequent renew and close calls
	Private *any `json:"privateData,omitempty"`
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

// Open executes the ephemeral resource open operation by calling the "open" method via JSON-RPC.
// It sends the configuration properties to the Deno runtime and retrieves the resource data.
//
// Parameters:
//   - ctx: The context for the operation, used for cancellation and timeouts
//   - params: The open request containing the ephemeral resource configuration properties
//
// Returns the open response containing the resource data and optional renewal time, or an error if the JSON-RPC call fails.
func (c *DenoClientEphemeralResource) Open(ctx context.Context, params *OpenRequest) (*OpenResponse, error) {
	var response *OpenResponse
	if err := c.Client.Socket.Call(ctx, "open", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call open method over JSON-RPC: %v", err)
	}
	return response, nil
}

// RenewRequest represents the request payload for renewing an ephemeral resource.
// It contains the private state data from the previous open or renew call.
type RenewRequest struct {
	// Private is the private state data from the previous open or renew response
	Private *any `json:"privateData,omitempty"`
}

// RenewResponse represents the response from renewing an ephemeral resource.
// It contains the updated renewal time and private state data.
type RenewResponse struct {
	// RenewAt is an optional Unix timestamp (in seconds) indicating when the resource should be renewed again
	RenewAt *int64 `json:"renewAt,omitempty"`
	// Private is optional updated private state data that will be passed to subsequent renew and close calls
	Private *any `json:"privateData,omitempty"`
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

// Renew executes the ephemeral resource renewal operation by calling the "renew" method via JSON-RPC.
// It sends the private state data to the Deno runtime to refresh the resource's lifetime.
//
// Parameters:
//   - ctx: The context for the operation, used for cancellation and timeouts
//   - params: The renew request containing the private state data
//
// Returns the renew response containing the next renewal time, or an error if the JSON-RPC call fails.
func (c *DenoClientEphemeralResource) Renew(ctx context.Context, params *RenewRequest) (*RenewResponse, error) {
	var response *RenewResponse
	if err := c.Client.Socket.Call(ctx, "renew", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call renew method over JSON-RPC: %v", err)
	}
	return response, nil
}

// CloseRequest represents the request payload for closing an ephemeral resource.
// It contains the private state data from the previous open or renew call.
type CloseRequest struct {
	// Private is the private state data from the previous open or renew response
	Private *any `json:"privateData,omitempty"`
}

// CloseResponse represents the response from closing an ephemeral resource.
// It indicates whether the close operation completed successfully.
type CloseResponse struct {
	// Done indicates whether the close operation completed successfully
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

// Close executes the ephemeral resource close operation by calling the "close" method via JSON-RPC.
// It sends the private state data to the Deno runtime to clean up the resource.
// Note: The close method is optional; if not implemented in the script, this method returns nil.
//
// Parameters:
//   - ctx: The context for the operation, used for cancellation and timeouts
//   - params: The close request containing the private state data
//
// Returns an error if the JSON-RPC call fails or the close operation is not complete.
// Returns nil if the close method is not implemented (CodeMethodNotFound).
func (c *DenoClientEphemeralResource) Close(ctx context.Context, params *CloseRequest) (*CloseResponse, error) {
	var response *CloseResponse
	if err := c.Client.Socket.Call(ctx, "close", params, &response); err != nil {

		// Close method is optional - return nil if not implemented
		var rpcErr *jsonrpc2.Error
		if errors.As(err, &rpcErr) && rpcErr.Code == jsonrpc2.CodeMethodNotFound {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to call close method over JSON-RPC: %v", err)
	}
	return response, nil
}
