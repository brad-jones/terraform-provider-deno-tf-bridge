package provider

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestDenoClient(t *testing.T) {
	// Enable test mode for direct stdout/stderr logging
	t.Setenv("DENO_TOFU_BRIDGE_TEST_MODE", "true")

	denoBinary, err := exec.LookPath("deno")
	assert.NoError(t, err)

	scriptPath := filepath.Join(t.TempDir(), "test.ts")
	err = os.WriteFile(scriptPath, []byte(`
		import { Hono } from "jsr:@hono/hono";

		const app = new Hono();

		app.get("/health", (c) => {
			return c.json({ status: "ok" });
		});

		app.post("/hello", async (c) => {
			const { name } = await c.req.json();
			return c.json({ message: "Hello " + name });
		});

		export default app satisfies Deno.ServeDefaultExport;
	`), 0644)
	assert.NoError(t, err)

	client := NewDenoClient(denoBinary, scriptPath, &denoPermissions{All: true})
	err = client.Start(t.Context())
	assert.NoError(t, err)
	defer func() {
		if err := client.Stop(); err != nil {
			t.Errorf("failed to stop client: %v", err)
		}
	}()

	var response *struct {
		Message string `json:"message"`
	}
	err = client.C().Post("/hello").SetBody(map[string]any{"name": "John Smith"}).Do(t.Context()).Into(&response)
	assert.NoError(t, err)
	assert.Equal(t, "Hello John Smith", response.Message)
}
