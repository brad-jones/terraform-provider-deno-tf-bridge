package jsocket

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/sourcegraph/jsonrpc2"
)

type StronglyTypedSocket struct {
	socket *JSocket
}

func NewTyped(ctx context.Context, reader io.ReadCloser, writer io.Writer, opts ...jsonrpc2.ConnOpt) *StronglyTypedSocket {
	self := &StronglyTypedSocket{}

	self.socket = New(ctx, reader, writer,
		TypedServerMethods(&StronglyTypedSocketServerMethods{self}),
		opts...,
	)

	return self
}

type HealthResponse struct {
	Ok bool `json:"ok"`
}

func (s *StronglyTypedSocket) Health(ctx context.Context) error {
	var response *HealthResponse
	if err := s.socket.Call(ctx, "health", nil, &response); err != nil {
		return fmt.Errorf("[Alice] Failed to call Bobs health method: %v", err)
	}
	if !response.Ok {
		return fmt.Errorf("[Alice] Bob is unhealthy")
	}
	return nil
}

type EchoRequest struct {
	Message string `json:"message"`
}

type EchoResponse struct {
	Echoed    string `json:"echoed"`
	Timestamp int64  `json:"timestamp"`
}

func (s *StronglyTypedSocket) Echo(ctx context.Context, params *EchoRequest) (*EchoResponse, error) {
	var response *EchoResponse
	if err := s.socket.Call(ctx, "echo", params, &response); err != nil {
		return nil, fmt.Errorf("[Alice] Failed to call Bobs echo method: %v", err)
	}
	return response, nil
}

type InvokeRequest struct {
	Count    int64 `json:"count"`
	DelaySec int64 `json:"delaySec"`
}

type InvokeResponse struct {
	ItemsProcessed int64 `json:"itemsProcessed"`
}

func (s *StronglyTypedSocket) Invoke(ctx context.Context, params *InvokeRequest) (*InvokeResponse, error) {
	var response *InvokeResponse
	if err := s.socket.Call(ctx, "invoke", params, &response); err != nil {
		return nil, fmt.Errorf("[Alice] Failed to call Bobs invoke method: %v", err)
	}
	return response, nil
}

func (s *StronglyTypedSocket) Shutdown(ctx context.Context) error {
	if err := s.socket.Notify(ctx, "shutdown", nil); err != nil {
		return fmt.Errorf("[Alice] Failed to notify Bobs shutdown method: %v", err)
	}
	return nil
}

func (s *StronglyTypedSocket) Close() error {
	return s.socket.Close()
}

type StronglyTypedSocketServerMethods struct {
	socket *StronglyTypedSocket
}

type InvokeProgressRequest struct {
	Msg string `json:"msg"`
}

func (s *StronglyTypedSocketServerMethods) InvokeProgress(ctx context.Context, params *InvokeProgressRequest) {
	log.Printf("invokeProgress: %s", params.Msg)
}
