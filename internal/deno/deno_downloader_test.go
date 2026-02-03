package deno

import (
	"context"
	"fmt"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/bitfield/script"
)

func TestGetDenoBinary(t *testing.T) {
	downloader := NewDenoDownloader()

	binPath, err := downloader.GetDenoBinary(context.Background(), "latest")
	assert.NoError(t, err)

	denoHelpText, err := script.Exec(fmt.Sprintf(`"%s" --help`, binPath)).String()
	assert.NoError(t, err)

	assert.Contains(t, denoHelpText, "A modern JavaScript and TypeScript runtime")
}
