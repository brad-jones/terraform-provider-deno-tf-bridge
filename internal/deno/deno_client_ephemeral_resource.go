package deno

import (
	"context"
	"errors"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
)

type DenoClientEphemeralResource struct {
	Client *DenoClient
}

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

type OpenRequest struct {
	Props any `json:"props"`
}

type OpenResponse struct {
	Result  any    `json:"result"`
	RenewAt *int64 `json:"renewAt,omitempty"`
	Private *any   `json:"privateData,omitempty"`
}

func (c *DenoClientEphemeralResource) Open(ctx context.Context, params *OpenRequest) (*OpenResponse, error) {
	var response *OpenResponse
	if err := c.Client.Socket.Call(ctx, "open", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call open method over JSON-RPC: %v", err)
	}
	return response, nil
}

type RenewRequest struct {
	Private *any `json:"privateData,omitempty"`
}

type RenewResponse struct {
	RenewAt *int64 `json:"renewAt,omitempty"`
	Private *any   `json:"privateData,omitempty"`
}

func (c *DenoClientEphemeralResource) Renew(ctx context.Context, params *RenewRequest) (*RenewResponse, error) {
	var response *RenewResponse
	if err := c.Client.Socket.Call(ctx, "renew", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call renew method over JSON-RPC: %v", err)
	}
	return response, nil
}

type CloseRequest struct {
	Private *any `json:"privateData,omitempty"`
}

type CloseResponse struct {
	Done bool `json:"done"`
}

func (c *DenoClientEphemeralResource) Close(ctx context.Context, params *CloseRequest) error {
	var response *CloseResponse
	if err := c.Client.Socket.Call(ctx, "close", params, &response); err != nil {

		// Close is method is optional
		var rpcErr *jsonrpc2.Error
		if errors.As(err, &rpcErr) && rpcErr.Code == jsonrpc2.CodeMethodNotFound {
			return nil
		}

		return fmt.Errorf("failed to call close method over JSON-RPC: %v", err)
	}
	if !response.Done {
		return fmt.Errorf("close call not done")
	}
	return nil
}
