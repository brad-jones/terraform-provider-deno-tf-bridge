package deno

import (
	"context"
	"fmt"

	"github.com/brad-jones/terraform-provider-denobridge/internal/jsocket"
	"github.com/hashicorp/terraform-plugin-framework/action"
)

type DenoClientAction struct {
	Client *DenoClient
}

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

type InvokeRequest struct {
	Props any `json:"props"`
}

type InvokeResponse struct {
	Done bool `json:"done"`
}

func (c *DenoClientAction) Invoke(ctx context.Context, params *InvokeRequest) error {
	var response *InvokeResponse
	if err := c.Client.Socket.Call(ctx, "invoke", params, &response); err != nil {
		return fmt.Errorf("failed to call invoke method over JSON-RPC: %v", err)
	}
	if !response.Done {
		return fmt.Errorf("invoke call not done")
	}
	return nil
}

type DenoClientActionServerMethods struct {
	resp *action.InvokeResponse
}

type InvokeProgressRequest struct {
	Message string `json:"message"`
}

func (c *DenoClientActionServerMethods) InvokeProgress(ctx context.Context, params *InvokeProgressRequest) {
	c.resp.SendProgress(action.InvokeProgressEvent{
		Message: params.Message,
	})
}
