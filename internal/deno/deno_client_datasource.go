package deno

import (
	"context"
	"fmt"
)

type DenoClientDatasource struct {
	Client *DenoClient
}

func NewDenoClientDatasource(denoBinaryPath, scriptPath, configPath string, permissions *Permissions) *DenoClientDatasource {
	return &DenoClientDatasource{
		NewDenoClient(
			denoBinaryPath,
			scriptPath,
			configPath,
			permissions,
			nil,
		),
	}
}

type ReadRequest struct {
	Props any `json:"props"`
}

type ReadResponse struct {
	Result any `json:"result"`
}

func (c *DenoClientDatasource) Read(ctx context.Context, params *ReadRequest) (*ReadResponse, error) {
	var response *ReadResponse
	if err := c.Client.Socket.Call(ctx, "read", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call read method over JSON-RPC: %v", err)
	}
	return response, nil
}
