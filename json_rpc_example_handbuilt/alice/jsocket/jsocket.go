// Package jsocket provides a simplified bidirectional JSON-RPC 2.0 client and server
// implementation built on top of the sourcegraph/jsonrpc2 library.
//
// JSocket enables easy creation of bidirectional RPC connections where both sides can
// act as both client and server simultaneously. It automatically handles request routing,
// method invocation via reflection, and supports flexible handler signatures.
//
// # Basic Usage
//
// Create a JSocket connection over any io.ReadWriter stream:
//
//	socket := jsocket.New(ctx, reader, writer, func(ctx context.Context, c *jsonrpc2.Conn) map[string]any {
//		return map[string]any{
//			"greet": func(params struct{ Name string }) (struct{ Message string }, error) {
//				return struct{ Message string }{
//					Message: "Hello, " + params.Name,
//				}, nil
//			},
//		}
//	})
//	defer socket.Close()
//
// Make RPC calls to the remote peer:
//
//	var result struct{ Message string }
//	err := socket.Call(ctx, "greet", struct{ Name string }{"Alice"}, &result)
//
// Send fire-and-forget notifications:
//
//	err := socket.Notify(ctx, "log", struct{ Level, Message string }{"info", "Started"})
//
// # Typed Server Methods
//
// For better type safety and organization, use TypedServerMethods to automatically
// convert a struct's methods into RPC handlers:
//
//	type MyServer struct {
//		// ... fields
//	}
//
//	func (s *MyServer) Greet(ctx context.Context, params *struct{ Name string }) (*struct{ Message string }, error) {
//		return &struct{ Message string }{
//			Message: "Hello, " + params.Name,
//		}, nil
//	}
//
//	func (s *MyServer) LogMessage(ctx context.Context, params *struct{ Level, Message string }) error {
//		// Handle notification
//		return nil
//	}
//
//	// Use it:
//	server := &MyServer{}
//	socket := jsocket.New(ctx, reader, writer, jsocket.TypedServerMethods(server))
//
// Method names are automatically converted from PascalCase to camelCase (e.g., Greet becomes greet).
//
// # Strongly Typed Client Wrappers
//
// For maximum type safety on the client side, you can create a wrapper struct that
// provides strongly-typed methods for calling remote procedures. This pattern combines
// type safety for both client calls and server handlers:
//
//	type MyClient struct {
//		socket *jsocket.JSocket
//	}
//
//	func NewMyClient(ctx context.Context, reader io.ReadCloser, writer io.Writer) *MyClient {
//		client := &MyClient{}
//
//		// Define server methods that this client will handle
//		client.socket = jsocket.New(ctx, reader, writer,
//			jsocket.TypedServerMethods(&MyServerMethods{client}))
//
//		return client
//	}
//
//	// Strongly-typed client method for calling remote "greet" RPC
//	func (c *MyClient) Greet(ctx context.Context, name string) (string, error) {
//		var result struct{ Message string }
//		err := c.socket.Call(ctx, "greet", struct{ Name string }{name}, &result)
//		if err != nil {
//			return "", err
//		}
//		return result.Message, nil
//	}
//
//	// Strongly-typed notification method
//	func (c *MyClient) SendLog(ctx context.Context, level, message string) error {
//		return c.socket.Notify(ctx, "log", struct{ Level, Message string }{level, message})
//	}
//
//	func (c *MyClient) Close() error {
//		return c.socket.Close()
//	}
//
//	// Server-side methods that handle incoming RPCs from the remote peer
//	type MyServerMethods struct {
//		client *MyClient
//	}
//
//	func (s *MyServerMethods) OnStatus(ctx context.Context, params *struct{ Status string }) {
//		// Handle incoming status notification from remote peer
//		log.Printf("Received status: %s", params.Status)
//
//		// Respond by making a client call back to the remote peer
//		// This demonstrates the bidirectional nature of the connection
//		if params.Status == "ready" {
//			greeting, err := s.client.Greet(ctx, "Server")
//			if err != nil {
//				log.Printf("Failed to greet: %v", err)
//			} else {
//				log.Printf("Got greeting: %s", greeting)
//			}
//		}
//	}
//
// This pattern provides compile-time type checking for all RPC interactions and creates
// a clean, documented API for your bidirectional communication protocol.
//
// # Handler Signatures
//
// Server methods can have flexible signatures:
//   - func(ctx context.Context, params *T) - no return value
//   - func(ctx context.Context, params *T) error - can fail
//   - func(ctx context.Context, params *T) (*R, error) - returns response
//   - func(ctx context.Context) ... - no parameters (for parameterless methods)
//
// Where T is the parameter type and R is the response type.
package jsocket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"unicode"

	"github.com/sourcegraph/jsonrpc2"
)

// JSocket is a bidirectional JSON-RPC 2.0 client and server wrapper.
// It provides a simplified interface for making remote procedure calls and
// handling incoming RPC requests over any io.ReadWriter stream.
// JSocket automatically routes incoming requests to registered server methods
// and supports both synchronous calls and fire-and-forget notifications.
type JSocket struct {
	conn *jsonrpc2.Conn
}

// New creates a new JSocket instance that wraps a JSON-RPC 2.0 bidirectional connection.
// It establishes a connection over the provided reader and writer streams, automatically
// routing incoming JSON-RPC requests to the appropriate server methods.
//
// The serverMethods parameter should return a map of method names to handler functions.
// Handler functions are invoked using reflection and can have flexible signatures:
//   - func(params T) - for methods that don't return values
//   - func(params T) error - for methods that may fail
//   - func(params T) (R, error) - for methods that return a response
//
// The ctx parameter is used for the lifetime of the connection. The connection will be
// closed when the context is cancelled.
//
// Additional connection options can be provided via opts to customize behavior such as
// logging, interceptors, or other JSON-RPC connection settings.
func New(ctx context.Context, reader io.ReadCloser, writer io.Writer, serverMethods func(ctx context.Context, c *jsonrpc2.Conn) map[string]any, opts ...jsonrpc2.ConnOpt) *JSocket {
	stream := jsonrpc2.NewPlainObjectStream(&struct {
		io.ReadCloser
		io.Writer
	}{
		ReadCloser: reader,
		Writer:     writer,
	})

	handler := jsonrpc2.AsyncHandler(
		jsonrpc2.HandlerWithError(func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) (any, error) {
			// Build the methods map
			methods := serverMethods(ctx, c)

			// Locate the method otherwise return a Not Found error
			method, ok := methods[r.Method]
			if !ok {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: "Method not found"}
			}

			// Call method with reflection
			methodValue := reflect.ValueOf(method)
			methodType := methodValue.Type()

			// Verify it's a function
			if methodType.Kind() != reflect.Func {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInternalError, Message: "Method is not a function"}
			}

			// Prepare arguments
			var args []reflect.Value

			// Check if method takes parameters
			if methodType.NumIn() > 0 {
				// Create a new instance of the parameter type
				paramType := methodType.In(0)
				paramValue := reflect.New(paramType)

				// Unmarshal params into the parameter if params exist
				if r.Params != nil && len(*r.Params) > 0 {
					if err := json.Unmarshal(*r.Params, paramValue.Interface()); err != nil {
						return nil, fmt.Errorf("failed to unmarshal params: %w", err)
					}
				}

				args = append(args, paramValue.Elem())
			}

			// Call the method
			results := methodValue.Call(args)

			// Process return values based on number of outputs
			switch methodType.NumOut() {
			case 0:
				// No return values
				return nil, nil
			case 1:
				// One return value - check if it's an error
				result := results[0]
				if result.Type().Implements(reflect.TypeFor[error]()) {
					if !result.IsNil() {
						return nil, fmt.Errorf("method failed: %w", result.Interface().(error))
					}
					return nil, nil
				}
				// Otherwise it's a response
				return result.Interface(), nil
			case 2:
				// Two return values - (response, error)
				response := results[0].Interface()
				errResult := results[1]
				if !errResult.IsNil() {
					return nil, fmt.Errorf("method failed: %w", errResult.Interface().(error))
				}
				return response, nil
			default:
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInternalError, Message: "Method has unsupported number of return values"}
			}
		}),
	)

	return &JSocket{jsonrpc2.NewConn(ctx, stream, handler, opts...)}
}

// Call sends a JSON-RPC request to the remote peer and waits for a response.
// The method parameter specifies the remote method to invoke, params contains the
// input parameters, and result will be populated with the response data.
// The call blocks until a response is received or the context is cancelled.
// Returns an error if the call fails or the remote method returns an error.
func (j *JSocket) Call(ctx context.Context, method string, params, result any, opts ...jsonrpc2.CallOption) error {
	return j.conn.Call(ctx, method, params, result, opts...)
}

// Notify sends a JSON-RPC notification to the remote peer without expecting a response.
// Notifications are fire-and-forget messages that don't include a request ID and won't
// receive a response from the server. This is useful for events or updates where no
// acknowledgment is needed.
func (j *JSocket) Notify(ctx context.Context, method string, params any, opts ...jsonrpc2.CallOption) error {
	return j.conn.Notify(ctx, method, params, opts...)
}

// Close closes the underlying JSON-RPC connection and releases associated resources.
// It should be called when the JSocket is no longer needed.
func (j *JSocket) Close() error {
	return j.conn.Close()
}

// TypedServerMethods converts a struct's exported methods into a map suitable for JSocket.
// It automatically converts method names from PascalCase to camelCase for JSON-RPC compatibility.
// Methods should have one of the following signatures:
//   - func(ctx context.Context, params *T)
//   - func(ctx context.Context, params *T) error
//   - func(ctx context.Context, params *T) (*R, error)
//
// Where T is the parameter type and R is the response type.
func TypedServerMethods(methods any) func(ctx context.Context, c *jsonrpc2.Conn) map[string]any {
	return func(ctx context.Context, c *jsonrpc2.Conn) map[string]any {
		methodsMap := make(map[string]any)

		val := reflect.ValueOf(methods)
		typ := val.Type()

		// Handle both struct and pointer to struct
		if typ.Kind() == reflect.Pointer {
			val = val.Elem()
			typ = val.Type()
		}

		if typ.Kind() != reflect.Struct {
			return methodsMap
		}

		// Get the pointer type for method lookup
		ptrType := reflect.PointerTo(typ)
		ptrVal := reflect.New(typ).Elem()
		ptrVal.Set(val)
		ptr := ptrVal.Addr()

		// Iterate through all methods
		for i := 0; i < ptrType.NumMethod(); i++ {
			method := ptrType.Method(i)

			// Skip unexported methods
			if !method.IsExported() {
				continue
			}

			// Convert method name to camelCase
			methodName := toCamelCase(method.Name)

			// Get the method bound to our instance
			methodFunc := ptr.Method(i)

			// Validate method signature
			methodType := methodFunc.Type()
			if !isValidServerMethod(methodType) {
				continue
			}

			// Create a wrapper that matches the expected signature for the handler
			// The handler expects: func(params T) or func(params T) (R, error) etc.
			wrapper := createMethodWrapper(methodFunc, methodType, ctx)
			methodsMap[methodName] = wrapper
		}

		return methodsMap
	}
}

// toCamelCase converts PascalCase to camelCase
func toCamelCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// isValidServerMethod checks if a method has a valid signature for server methods
func isValidServerMethod(methodType reflect.Type) bool {
	// Must have at least context.Context as first parameter
	if methodType.NumIn() < 1 {
		return false
	}

	// First parameter should be context.Context
	if !methodType.In(0).Implements(reflect.TypeFor[context.Context]()) {
		return false
	}

	// Can have 1 (ctx only) or 2 (ctx + params) input parameters
	if methodType.NumIn() > 2 {
		return false
	}

	// Can have 0, 1, or 2 return values
	// 0: func(ctx, params)
	// 1: func(ctx, params) error or func(ctx, params) response
	// 2: func(ctx, params) (response, error)
	if methodType.NumOut() > 2 {
		return false
	}

	return true
}

// createMethodWrapper creates a wrapper function that strips the context parameter
// since the handler will inject it
func createMethodWrapper(methodFunc reflect.Value, methodType reflect.Type, ctx context.Context) any {
	// Determine the wrapper type based on the method signature
	numIn := methodType.NumIn()
	numOut := methodType.NumOut()

	if numIn == 1 {
		// Method only takes context: func(ctx context.Context) ...
		if numOut == 0 {
			// func(ctx context.Context)
			return func() {
				methodFunc.Call([]reflect.Value{reflect.ValueOf(ctx)})
			}
		} else if numOut == 1 {
			// func(ctx context.Context) error or func(ctx context.Context) response
			return func() any {
				results := methodFunc.Call([]reflect.Value{reflect.ValueOf(ctx)})
				return results[0].Interface()
			}
		} else {
			// func(ctx context.Context) (response, error)
			return func() (any, error) {
				results := methodFunc.Call([]reflect.Value{reflect.ValueOf(ctx)})
				if !results[1].IsNil() {
					return nil, results[1].Interface().(error)
				}
				return results[0].Interface(), nil
			}
		}
	} else {
		// Method takes context + params: func(ctx context.Context, params T) ...
		paramType := methodType.In(1)

		if numOut == 0 {
			// func(ctx context.Context, params T)
			return reflect.MakeFunc(
				reflect.FuncOf([]reflect.Type{paramType}, []reflect.Type{}, false),
				func(args []reflect.Value) []reflect.Value {
					methodFunc.Call([]reflect.Value{reflect.ValueOf(ctx), args[0]})
					return []reflect.Value{}
				},
			).Interface()
		} else if numOut == 1 {
			// func(ctx context.Context, params T) error or func(ctx context.Context, params T) response
			returnType := methodType.Out(0)
			return reflect.MakeFunc(
				reflect.FuncOf([]reflect.Type{paramType}, []reflect.Type{returnType}, false),
				func(args []reflect.Value) []reflect.Value {
					return methodFunc.Call([]reflect.Value{reflect.ValueOf(ctx), args[0]})
				},
			).Interface()
		} else {
			// func(ctx context.Context, params T) (response, error)
			returnType := methodType.Out(0)
			errorType := methodType.Out(1)
			return reflect.MakeFunc(
				reflect.FuncOf([]reflect.Type{paramType}, []reflect.Type{returnType, errorType}, false),
				func(args []reflect.Value) []reflect.Value {
					return methodFunc.Call([]reflect.Value{reflect.ValueOf(ctx), args[0]})
				},
			).Interface()
		}
	}
}
