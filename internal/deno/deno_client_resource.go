package deno

import (
	"context"
	"errors"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
)

type DenoClientResource struct {
	Client *DenoClient
}

func NewDenoClientResource(denoBinaryPath, scriptPath, configPath string, permissions *Permissions) *DenoClientResource {
	return &DenoClientResource{
		NewDenoClient(
			denoBinaryPath,
			scriptPath,
			configPath,
			permissions,
			nil,
		),
	}
}

type CreateRequest struct {
	Props any `json:"props"`
}

type CreateResponse struct {
	ID    string `json:"id"`
	State any    `json:"state"`
}

func (c *DenoClientResource) Create(ctx context.Context, params *CreateRequest) (*CreateResponse, error) {
	var response *CreateResponse
	if err := c.Client.Socket.Call(ctx, "create", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call create method over JSON-RPC: %v", err)
	}
	return response, nil
}

type CreateReadRequest struct {
	ID    string `json:"id"`
	Props any    `json:"props"`
}

type CreateReadResponse struct {
	Props  *any  `json:"props"`
	State  *any  `json:"state"`
	Exists *bool `json:"exists"`
}

func (c *DenoClientResource) Read(ctx context.Context, params *CreateReadRequest) (*CreateReadResponse, error) {
	var response *CreateReadResponse
	if err := c.Client.Socket.Call(ctx, "read", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call read method over JSON-RPC: %v", err)
	}
	return response, nil
}

type UpdateRequest struct {
	ID           string `json:"id"`
	NextProps    any    `json:"nextProps"`
	CurrentProps any    `json:"currentProps"`
	CurrentState any    `json:"currentState"`
}

type UpdateResponse struct {
	State *any `json:"state"`
}

func (c *DenoClientResource) Update(ctx context.Context, params *UpdateRequest) (*UpdateResponse, error) {
	var response *UpdateResponse
	if err := c.Client.Socket.Call(ctx, "update", params, &response); err != nil {
		return nil, fmt.Errorf("failed to call update method over JSON-RPC: %v", err)
	}
	return response, nil
}

type DeleteRequest struct {
	ID    string `json:"id"`
	Props any    `json:"props"`
	State any    `json:"state"`
}

type DeleteResponse struct {
	Done bool `json:"done"`
}

func (c *DenoClientResource) Delete(ctx context.Context, params *DeleteRequest) error {
	var response *DeleteResponse
	if err := c.Client.Socket.Call(ctx, "delete", params, &response); err != nil {
		return fmt.Errorf("failed to call delete method over JSON-RPC: %v", err)
	}
	if !response.Done {
		return fmt.Errorf("delete not done")
	}
	return nil
}

type ModifyPlanRequest struct {
	ID           *string `json:"id,omitempty"`
	PlanType     string  `json:"planType"`
	NextProps    any     `json:"nextProps"`
	CurrentProps any     `json:"currentProps,omitempty"`
	CurrentState any     `json:"currentState,omitempty"`
}

type ModifyPlanResponse struct {
	NoChanges           *bool `json:"noChanges,omitempty"`
	ModifiedProps       *any  `json:"modifiedProps,omitempty"`
	RequiresReplacement *bool `json:"requiresReplacement,omitempty"`
	Diagnostics         *[]struct {
		Severity string  `json:"severity"`
		Summary  string  `json:"summary"`
		Detail   string  `json:"detail"`
		PropName *string `json:"propName,omitempty"`
	} `json:"diagnostics,omitempty"`
}

func (c *DenoClientResource) ModifyPlan(ctx context.Context, params *ModifyPlanRequest) (*ModifyPlanResponse, error) {
	var response *ModifyPlanResponse
	if err := c.Client.Socket.Call(ctx, "modifyPlan", params, &response); err != nil {

		// ModifyPlan method is optional
		var rpcErr *jsonrpc2.Error
		if errors.As(err, &rpcErr) && rpcErr.Code == jsonrpc2.CodeMethodNotFound {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to call modifyPlan method over JSON-RPC: %v", err)
	}

	return response, nil
}
