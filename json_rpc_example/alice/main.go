package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"
)

// Result types matching Bob's OpenRPC spec
type HealthResult struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

type EchoResult struct {
	Echoed    string `json:"echoed"`
	Timestamp int64  `json:"timestamp"`
}

type ProcessResult struct {
	Processed int      `json:"processed"`
	Results   []string `json:"results"`
	Duration  int64    `json:"duration"`
}

type ShutdownResult struct {
	Message string `json:"message"`
}

// Notification types from Bob
type ProgressParams struct {
	Message string  `json:"message"`
	Percent float64 `json:"percent"`
}

type LogParams struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// JSON-RPC 2.0 message structures
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// RPCClient manages JSON-RPC communication over stdin/stdout
type RPCClient struct {
	stdin       io.Writer
	stdout      io.Reader
	mu          sync.Mutex
	nextID      int
	pending     map[int]chan *JSONRPCResponse
	notifyFunc  func(method string, params interface{})
	requestFunc func(method string, params interface{}) (interface{}, error)
}

func NewRPCClient(stdin io.Writer, stdout io.Reader, notifyFunc func(method string, params interface{}), requestFunc func(method string, params interface{}) (interface{}, error)) *RPCClient {
	return &RPCClient{
		stdin:       stdin,
		stdout:      stdout,
		nextID:      1,
		pending:     make(map[int]chan *JSONRPCResponse),
		notifyFunc:  notifyFunc,
		requestFunc: requestFunc,
	}
}

func (c *RPCClient) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	c.mu.Lock()
	id := c.nextID
	c.nextID++
	respChan := make(chan *JSONRPCResponse, 1)
	c.pending[id] = respChan
	c.mu.Unlock()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}

	if _, err := c.stdin.Write(append(reqBytes, '\n')); err != nil {
		return err
	}

	select {
	case resp := <-respChan:
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()

		if resp.Error != nil {
			return &RPCError{Code: resp.Error.Code, Message: resp.Error.Message}
		}

		if result != nil && resp.Result != nil {
			resultBytes, err := json.Marshal(resp.Result)
			if err != nil {
				return err
			}
			return json.Unmarshal(resultBytes, result)
		}
		return nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return ctx.Err()
	}
}

func (c *RPCClient) handleIncoming(scanner *bufio.Scanner) {
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("[Alice] Error parsing message: %v", err)
			continue
		}

		// Check if it has a method field (request or notification)
		if method, hasMethod := msg["method"].(string); hasMethod {
			// Check if it's a notification (no id field) or a request (has id)
			if msgID, hasID := msg["id"]; !hasID {
				// It's a notification
				params := msg["params"]
				if c.notifyFunc != nil {
					c.notifyFunc(method, params)
				}
			} else {
				// It's a request from Bob - handle it and send response
				params := msg["params"]
				go func() {
					var resp JSONRPCResponse
					resp.JSONRPC = "2.0"
					resp.ID = msgID

					if c.requestFunc != nil {
						result, err := c.requestFunc(method, params)
						if err != nil {
							resp.Error = &JSONRPCError{
								Code:    -32603,
								Message: err.Error(),
							}
						} else {
							resp.Result = result
						}
					} else {
						resp.Error = &JSONRPCError{
							Code:    -32601,
							Message: "Method not found",
						}
					}

					respBytes, _ := json.Marshal(resp)
					c.stdin.Write(append(respBytes, '\n'))
				}()
			}
			continue
		}

		// It's a response to one of our requests
		var resp JSONRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			log.Printf("[Alice] Error parsing response: %v", err)
			continue
		}

		c.mu.Lock()
		if ch, ok := c.pending[int(resp.ID.(float64))]; ok {
			ch <- &resp
		}
		c.mu.Unlock()
	}
}

type RPCError struct {
	Code    int
	Message string
}

func (e *RPCError) Error() string {
	return e.Message
}

func handleNotification(method string, params interface{}) {
	switch method {
	case "progress":
		paramsBytes, _ := json.Marshal(params)
		var p ProgressParams
		if err := json.Unmarshal(paramsBytes, &p); err == nil {
			log.Printf("[Alice] Progress: %.0f%% - %s", p.Percent, p.Message)
		}
	case "log":
		paramsBytes, _ := json.Marshal(params)
		var p LogParams
		if err := json.Unmarshal(paramsBytes, &p); err == nil {
			log.Printf("[Alice] [%s] %s", p.Level, p.Message)
		}
	default:
		log.Printf("[Alice] Unknown notification: %s", method)
	}
}

func handleRequest(method string, params interface{}) (interface{}, error) {
	switch method {
	case "getConfig":
		// Bob can request configuration from Alice
		log.Printf("[Alice] Bob requested configuration")
		return map[string]interface{}{
			"maxRetries":  3,
			"timeout":     30,
			"enableDebug": true,
			"environment": "production",
		}, nil
	case "shouldContinue":
		// Bob can ask if processing should continue
		log.Printf("[Alice] Bob asked if processing should continue")
		return map[string]interface{}{
			"continue": true,
			"reason":   "All systems operational",
		}, nil
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func main() {
	log.Println("[Alice] Starting Alice (Go parent process)")

	// Start Bob as a child process
	cmd := exec.Command("deno", "run", "-qA", "bob/main.ts")

	// Get pipes for stdin/stdout
	bobStdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("[Alice] Failed to create stdin pipe: %v", err)
	}

	bobStdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("[Alice] Failed to create stdout pipe: %v", err)
	}

	bobStderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("[Alice] Failed to create stderr pipe: %v", err)
	}

	// Start Bob
	if err := cmd.Start(); err != nil {
		log.Fatalf("[Alice] Failed to start Bob: %v", err)
	}
	log.Println("[Alice] Started Bob (Deno child process)")

	// Forward Bob's stderr to our stderr
	go func() {
		scanner := bufio.NewScanner(bobStderr)
		for scanner.Scan() {
			log.Printf("[Bob stderr] %s", scanner.Text())
		}
	}()

	// Create RPC client
	client := NewRPCClient(bobStdin, bobStdout, handleNotification, handleRequest)

	// Start reading responses in background
	scanner := bufio.NewScanner(bobStdout)
	go client.handleIncoming(scanner)

	// Give Bob a moment to initialize
	time.Sleep(500 * time.Millisecond)

	ctx := context.Background()

	// Test 1: Health check
	log.Println("\n[Alice] === Test 1: Health Check ===")
	var healthResult HealthResult
	if err := client.Call(ctx, "health", nil, &healthResult); err != nil {
		log.Fatalf("[Alice] Health check failed: %v", err)
	}
	log.Printf("[Alice] Health check passed: status=%s, timestamp=%d", healthResult.Status, healthResult.Timestamp)

	// Test 2: Echo message
	log.Println("\n[Alice] === Test 2: Echo Message ===")
	echoParams := map[string]interface{}{
		"message": "Hello from Alice!",
	}
	var echoResult EchoResult
	if err := client.Call(ctx, "echo", echoParams, &echoResult); err != nil {
		log.Fatalf("[Alice] Echo failed: %v", err)
	}
	log.Printf("[Alice] Echo result: %s (timestamp: %d)", echoResult.Echoed, echoResult.Timestamp)

	// Test 3: Process with progress (demonstrates streaming notifications)
	log.Println("\n[Alice] === Test 3: Process with Progress (Streaming) ===")
	processParams := map[string]interface{}{
		"items":   []string{"apple", "banana", "cherry", "date", "elderberry"},
		"delayMs": 200,
	}
	var processResult ProcessResult
	if err := client.Call(ctx, "processWithProgress", processParams, &processResult); err != nil {
		log.Fatalf("[Alice] Process with progress failed: %v", err)
	}
	log.Printf("[Alice] Processing complete: processed=%d items in %dms", processResult.Processed, processResult.Duration)
	log.Printf("[Alice] Results: %v", processResult.Results)

	// Test 4: Graceful shutdown
	log.Println("\n[Alice] === Test 4: Graceful Shutdown ===")
	var shutdownResult ShutdownResult
	if err := client.Call(ctx, "shutdown", nil, &shutdownResult); err != nil {
		log.Printf("[Alice] Shutdown call error (expected if Bob exits quickly): %v", err)
	} else {
		log.Printf("[Alice] Shutdown response: %s", shutdownResult.Message)
	}

	// Give Bob time to exit
	time.Sleep(200 * time.Millisecond)

	// Wait for Bob to exit
	if err := cmd.Wait(); err != nil {
		// Exit code 0 is expected, anything else is an error
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 0 {
				log.Printf("[Alice] Bob exited with code: %d", exitErr.ExitCode())
			}
		}
	}

	log.Println("\n[Alice] === All tests completed successfully ===")
}
