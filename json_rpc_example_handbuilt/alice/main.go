package main

import (
	"bufio"
	"context"
	"log"
	"os/exec"

	"github.com/brad-jones/terraform-provider-denobridge/json_rpc_example_handbuilt/alice/jsocket"
	"github.com/sourcegraph/jsonrpc2"
)

func main() {
	ctx := context.Background()

	log.Println("[Alice] Starting Alice (Go parent process)")

	// Create Bob as a child process
	cmd := exec.CommandContext(ctx, "deno", "run", "-qA", "bob/main.ts")

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
			log.Printf("[Bob] %s", scanner.Text())
		}
	}()

	// Create a new JSON RPC socket
	socket := jsocket.NewTyped(ctx, bobStdout, bobStdin, jsonrpc2.LogMessages(log.Default()))
	defer socket.Close()

	// Wait for Bob to become healthy
	if err := socket.Health(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("[Alice] Bob is healthy")

	// Call a method on Bob
	echoResponse, err := socket.Echo(ctx, &jsocket.EchoRequest{Message: "Hello Bob"})
	if err != nil {
		log.Fatal(err)
	}
	if echoResponse.Echoed != "Hello Bob" {
		log.Fatalf("[Alice] Bob did not echo my message to him as expected")
	}
	log.Println("[Alice] Bob return my message correctly")

	// Call the invoke method and receive progress updates
	invokeResponse, err := socket.Invoke(ctx, &jsocket.InvokeRequest{Count: 3, DelaySec: 1})
	if err != nil {
		log.Fatal(err)
	}
	if invokeResponse.ItemsProcessed != 3 {
		log.Fatalf("[Alice] Bob did not process all my invoke requests as expected")
	}
	log.Println("[Alice] Bob has finished processing my invoke call")

	// Shutdown bob gracefully
	if err := socket.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}

	// Wait for Bob to exit
	if err := cmd.Wait(); err != nil {
		// Exit code 0 is expected, anything else is an error
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 0 {
				log.Printf("[Alice] Bob exited with code: %d", exitErr.ExitCode())
			}
		}
	}

	log.Printf("[Alice] Bob has shutdown, example finished")
}
