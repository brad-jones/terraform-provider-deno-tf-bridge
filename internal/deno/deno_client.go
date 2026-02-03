package deno

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/brad-jones/terraform-provider-denobridge/internal/jsocket"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/sourcegraph/jsonrpc2"
)

// DenoClient manages a Deno child process and communication via JSON-RPC with it.
type DenoClient struct {
	ctx            context.Context
	scriptPath     string
	configPath     string
	permissions    *Permissions
	denoBinaryPath string
	process        *exec.Cmd
	rpcMethods     func(ctx context.Context, c *jsonrpc2.Conn) map[string]any
	Socket         *jsocket.JSocket
}

// NewDenoClient creates a new Deno client for the given script.
func NewDenoClient(denoBinaryPath, scriptPath, configPath string, permissions *Permissions, rpcMethods func(ctx context.Context, c *jsonrpc2.Conn) map[string]any) *DenoClient {
	return &DenoClient{
		scriptPath:     scriptPath,
		configPath:     configPath,
		permissions:    permissions,
		denoBinaryPath: denoBinaryPath,
		rpcMethods:     rpcMethods,
	}
}

// Start launches the Deno JSON-RPC process.
func (c *DenoClient) Start(ctx context.Context) error {
	// Store context for logging
	c.ctx = ctx

	// Build Deno command arguments
	args := []string{"run", "-q"}

	// Attempt to locate a deno config file if none given
	configPath := c.configPath
	if configPath == "" {
		configPath = locateDenoConfigFile(c.scriptPath)
	}
	if configPath != "" && configPath != "/dev/null" {
		args = append(args, "-c", configPath)
	}

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

	// Handle script path - support file:// URLs and remote URLs
	var scriptArg string
	if strings.Contains(c.scriptPath, "://") {
		// Parse URL
		parsedURL, err := url.Parse(c.scriptPath)
		if err != nil {
			return fmt.Errorf("failed to parse script URL: %w", err)
		}

		if parsedURL.Scheme == "file" {
			// Convert file:// URL to local path
			path := parsedURL.Path
			// On Windows, url.Parse for file:///C:/path gives Path="/C:/path"
			// We need to remove the leading slash before the drive letter
			if len(path) > 2 && path[0] == '/' && path[2] == ':' {
				path = path[1:]
			}
			localPath := filepath.FromSlash(path)
			absPath, err := filepath.Abs(localPath)
			if err != nil {
				return fmt.Errorf("failed to resolve script path: %w", err)
			}
			scriptArg = absPath
		} else {
			// Remote URL (http://, https://, etc.) - pass as-is
			scriptArg = c.scriptPath
		}
	} else {
		// Local file path - convert to absolute path
		absPath, err := filepath.Abs(c.scriptPath)
		if err != nil {
			return fmt.Errorf("failed to resolve script path: %w", err)
		}
		scriptArg = absPath
	}
	args = append(args, scriptArg)

	// Create command
	c.process = exec.CommandContext(ctx, c.denoBinaryPath, args...)

	// Log the full command being executed
	fullCmd := append([]string{c.denoBinaryPath}, args...)
	cmdStr := strings.Join(fullCmd, " ")
	if isTestContext() {
		log.Printf("[DEBUG] Executing Deno command: %s", cmdStr)
	} else {
		tflog.Debug(ctx, fmt.Sprintf("Executing Deno command: %s", cmdStr))
	}

	// Get pipes to the child proc stdio
	stdin, err := c.process.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

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

	// Pipe stderr to tflog
	go pipeToDebugLog(ctx, stderr, "[deno stderr] ")

	// Create the jsocket
	c.Socket = jsocket.New(ctx, stdout, stdin, c.rpcMethods)

	// Wait for the server to be ready
	var response struct {
		Ok bool `json:"ok"`
	}
	if err := c.Socket.Call(ctx, "health", nil, &response); err != nil {
		return fmt.Errorf("failed to call the Deno JSON-RPC servers health method: %w", err)
	}
	if !response.Ok {
		return fmt.Errorf("deno process unhealthy: %w", err)
	}

	return nil
}

// Stop terminates the Deno child process.
func (c *DenoClient) Stop() error {
	if c.Socket != nil {
		if err := c.Socket.Notify(c.ctx, "shutdown", nil); err != nil {
			return fmt.Errorf("failed to notify deno child proc to shutdown gracefully: %v", err)
		}
		if err := c.Socket.Close(); err != nil {
			return fmt.Errorf("failed to close jsocket and release resources: %w", err)
		}
	}
	if c.process != nil {
		if err := c.process.Wait(); err != nil {
			return fmt.Errorf("deno child proc died: %w", err)
		}
	}
	return nil
}

// isTestContext returns true if running in a test context.
func isTestContext() bool {
	// Check if TF_LOG_PROVIDER_DENO_TOFU_BRIDGE is not set (typical in tests)
	// or if explicit test mode is enabled
	return os.Getenv("DENO_TOFU_BRIDGE_TEST_MODE") == "true"
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

// cachedConfigLookups stores config file paths to avoid repeated filesystem lookups.
var cachedConfigLookups = make(map[string]string)

// locateDenoConfigFile searches for a Deno configuration file (deno.json or deno.jsonc)
// starting from the script file's directory and traversing upward through parent
// directories until found or root is reached.
//
// Accepts both regular file paths and file:// URLs.
// Results are cached to avoid repeated filesystem operations for the same file paths.
func locateDenoConfigFile(scriptPath string) string {
	// Convert file URL to path if needed
	if strings.HasPrefix(scriptPath, "file://") {
		parsedURL, err := url.Parse(scriptPath)
		if err == nil && parsedURL.Scheme == "file" {
			// On Windows, url.Parse for file:///C:/path gives Path="/C:/path"
			// We need to remove the leading slash before the drive letter
			path := parsedURL.Path
			if len(path) > 2 && path[0] == '/' && path[2] == ':' {
				path = path[1:]
			}
			scriptPath = filepath.FromSlash(path)
		}
	}

	// Check if scriptPath has a protocol scheme other than file://
	// If so, return empty string as remote script loading is not supported
	if strings.Contains(scriptPath, "://") {
		return ""
	}

	// Check cache first
	if cached, ok := cachedConfigLookups[scriptPath]; ok {
		return cached
	}

	// Start from the directory containing the script
	currentDir := filepath.Dir(scriptPath)
	volumeName := filepath.VolumeName(currentDir)

	// Walk up the directory tree
	for {
		// Check for deno.json
		denoJsonPath := filepath.Join(currentDir, "deno.json")
		if _, err := os.Stat(denoJsonPath); err == nil {
			cachedConfigLookups[scriptPath] = denoJsonPath
			return denoJsonPath
		}

		// Check for deno.jsonc
		denoJsoncPath := filepath.Join(currentDir, "deno.jsonc")
		if _, err := os.Stat(denoJsoncPath); err == nil {
			cachedConfigLookups[scriptPath] = denoJsoncPath
			return denoJsoncPath
		}

		// Get parent directory
		parentDir := filepath.Dir(currentDir)

		// Check if we've reached the root
		// On Windows: "C:\" becomes "C:\", on Unix: "/" becomes "/"
		if parentDir == currentDir || parentDir == volumeName || parentDir == string(filepath.Separator) {
			break
		}

		currentDir = parentDir
	}

	// No config file found
	return ""
}
