package provider

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/imroc/req/v3"
)

// DenoClient manages a Deno HTTP server process and communication with it.
type DenoClient struct {
	scriptPath     string
	permissions    *denoPermissions
	denoBinaryPath string
	process        *exec.Cmd
	port           int
	baseURL        string
	ctx            context.Context
}

// NewDenoClient creates a new Deno client for the given script.
func NewDenoClient(denoBinaryPath string, scriptPath string, permissions *denoPermissions) *DenoClient {
	return &DenoClient{
		scriptPath:     scriptPath,
		permissions:    permissions,
		denoBinaryPath: denoBinaryPath,
	}
}

// Start launches the Deno HTTP server process.
func (c *DenoClient) Start(ctx context.Context) error {
	// Store context for logging
	c.ctx = ctx

	// Find an available port
	port, err := getAvailablePort()
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}
	c.port = port
	c.baseURL = fmt.Sprintf("http://127.0.0.1:%d", port)

	// Build Deno command arguments
	args := []string{"serve", "-q", "--port", fmt.Sprintf("%d", port)}

	// Add permissions
	if c.permissions != nil {
		if c.permissions.All {
			args = append(args, "--allow-all")
		} else {
			for _, perm := range c.permissions.Allow {
				args = append(args, fmt.Sprintf("--allow-%s", perm))
			}
			for _, perm := range c.permissions.Deny {
				args = append(args, fmt.Sprintf("--deny-%s", perm))
			}
		}
	}

	// Add script path
	absPath, err := filepath.Abs(c.scriptPath)
	if err != nil {
		return fmt.Errorf("failed to resolve script path: %w", err)
	}
	args = append(args, absPath)

	// Create command
	c.process = exec.CommandContext(ctx, c.denoBinaryPath, args...)

	// Capture stdout and stderr through tflog
	stdout, err := c.process.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := c.process.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := c.process.Start(); err != nil {
		return fmt.Errorf("failed to start Deno process: %w", err)
	}

	// Start goroutines to pipe output to tflog
	go pipeToDebugLog(ctx, stdout, "[deno stdout] ")
	go pipeToErrorLog(ctx, stderr, "[deno stderr] ")

	// Wait for the server to be ready
	if err := c.waitForReady(ctx, 30*time.Second); err != nil {
		if stopErr := c.Stop(); stopErr != nil {
			return fmt.Errorf("deno server failed to become ready: %w, and failed to stop: %w", err, stopErr)
		}
		return fmt.Errorf("deno server failed to become ready: %w", err)
	}

	return nil
}

// Stop terminates the Deno HTTP server process.
func (c *DenoClient) Stop() error {
	if c.process != nil && c.process.Process != nil {
		if err := c.process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill Deno process: %w", err)
		}
	}
	return nil
}

// waitForReady polls the health endpoint until the server responds.
func (c *DenoClient) waitForReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Monitor process exit in a goroutine
	processExited := make(chan error, 1)
	go func() {
		if c.process != nil {
			err := c.process.Wait()
			if err != nil {
				processExited <- fmt.Errorf("deno process exited with error: %w", err)
			} else {
				processExited <- fmt.Errorf("deno process exited unexpectedly")
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Deno server to start")
		case err := <-processExited:
			return err
		case <-ticker.C:
			resp, err := c.C().R().SetContext(ctx).Get("/health")
			if err != nil {
				continue
			}
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
				return nil
			}
		}
	}
}

// C returns a new req client instance, configured to talk to the deno child process. see: https://req.cool/
func (c *DenoClient) C() *req.Client {
	client := req.C().
		SetBaseURL(c.baseURL).
		SetCommonContentType("application/json").
		SetLogger(&tflogAdapter{ctx: c.ctx})

	// Only enable debug logging and dumping if TF_LOG is set to DEBUG
	if os.Getenv("TF_LOG") == "DEBUG" {
		client = client.EnableDebugLog().DevMode()
	}

	return client
}

// getAvailablePort finds an available port on localhost.
func getAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("failed to get TCP address from listener")
	}
	return tcpAddr.Port, nil
}

// isTestContext returns true if running in a test context.
func isTestContext() bool {
	// Check if TF_LOG_PROVIDER_DENO_TOFU_BRIDGE is not set (typical in tests)
	// or if explicit test mode is enabled
	return os.Getenv("DENO_TOFU_BRIDGE_TEST_MODE") == "true"
}

// tflogAdapter adapts tflog to the req logger interface.
type tflogAdapter struct {
	ctx context.Context
}

func (l *tflogAdapter) Debugf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	// Remove trailing newlines as tflog adds them
	msg = strings.TrimRight(msg, "\n")
	if isTestContext() {
		log.Printf("[DEBUG] [req] %s", msg)
	} else {
		tflog.Debug(l.ctx, "[req] "+msg)
	}
}

func (l *tflogAdapter) Warnf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	msg = strings.TrimRight(msg, "\n")
	if isTestContext() {
		log.Printf("[WARN] [req] %s", msg)
	} else {
		tflog.Warn(l.ctx, "[req] "+msg)
	}
}

func (l *tflogAdapter) Errorf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	msg = strings.TrimRight(msg, "\n")
	if isTestContext() {
		log.Printf("[ERROR] [req] %s", msg)
	} else {
		tflog.Error(l.ctx, "[req] "+msg)
	}
}

// pipeToDebugLog reads from a reader and logs each line as debug.
func pipeToDebugLog(ctx context.Context, reader io.Reader, prefix string) {
	scanner := bufio.NewScanner(reader)
	if isTestContext() {
		// In test context, write directly to stdout
		for scanner.Scan() {
			log.Printf("[DEBUG] %s%s", prefix, scanner.Text())
		}
	} else {
		// In Terraform context, use tflog
		for scanner.Scan() {
			tflog.Debug(ctx, prefix+scanner.Text())
		}
	}
}

// pipeToErrorLog reads from a reader and logs each line as error.
func pipeToErrorLog(ctx context.Context, reader io.Reader, prefix string) {
	scanner := bufio.NewScanner(reader)
	if isTestContext() {
		// In test context, write directly to stderr
		for scanner.Scan() {
			log.Printf("[ERROR] %s%s", prefix, scanner.Text())
		}
	} else {
		// In Terraform context, use tflog
		for scanner.Scan() {
			tflog.Error(ctx, prefix+scanner.Text())
		}
	}
}
