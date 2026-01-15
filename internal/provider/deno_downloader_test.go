package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/bitfield/script"
)

const githubAPITimeout = 3 * time.Second

func TestGetDenoBinary(t *testing.T) {
	// Skip this test if we're in an offline environment (e.g., CI with network restrictions)
	// This test requires downloading Deno from GitHub which may not be available
	if !canAccessGitHub() {
		t.Skip("Skipping TestGetDenoBinary: GitHub is not accessible (likely offline CI environment)")
	}

	downloader := NewDenoDownloader()

	binPath, err := downloader.GetDenoBinary(context.Background(), "latest")
	assert.NoError(t, err)

	denoHelpText, err := script.Exec(fmt.Sprintf(`"%s" --help`, binPath)).String()
	assert.NoError(t, err)

	assert.Contains(t, denoHelpText, "A modern JavaScript and TypeScript runtime")
}

// canAccessGitHub checks if the GitHub API is accessible.
func canAccessGitHub() bool {
	ctx, cancel := context.WithTimeout(context.Background(), githubAPITimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com", nil)
	if err != nil {
		return false
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}
