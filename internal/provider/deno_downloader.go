package provider

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	maxVersionsToKeep = 3
	githubAPIBase     = "https://api.github.com"
	denoRepo          = "denoland/deno"
)

// DenoDownloader manages downloading and caching Deno binaries.
type DenoDownloader struct {
	mu sync.Mutex
}

// githubRelease represents a GitHub release response.
type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

// githubAsset represents a GitHub release asset.
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

// NewDenoDownloader creates a new Deno downloader.
func NewDenoDownloader() *DenoDownloader {
	return &DenoDownloader{}
}

// GetDenoBinary returns the path to a Deno binary for the specified version.
// It checks the cache first, and downloads if necessary.
// version can be "latest" or a specific version like "v2.1.4".
func (d *DenoDownloader) GetDenoBinary(ctx context.Context, version string) (string, error) {
	// Lock to prevent concurrent downloads
	d.mu.Lock()
	defer d.mu.Unlock()

	// Get the cache directory
	cacheDir, err := d.getCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get cache directory: %w", err)
	}

	// Resolve version if "latest"
	resolvedVersion := version
	if version == "latest" {
		tflog.Info(ctx, "Resolving latest Deno version")
		resolved, err := d.getLatestVersion(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to resolve latest version: %w", err)
		}
		resolvedVersion = resolved
		tflog.Info(ctx, fmt.Sprintf("Resolved latest version to %s", resolvedVersion))
	}

	// Check if binary already exists in cache
	binaryPath := filepath.Join(cacheDir, resolvedVersion, denoBinaryName())
	if _, err := os.Stat(binaryPath); err == nil {
		tflog.Info(ctx, fmt.Sprintf("Using cached Deno binary at %s", binaryPath))
		return binaryPath, nil
	}

	// Download and install the binary
	tflog.Info(ctx, fmt.Sprintf("Downloading Deno version %s", resolvedVersion))
	if err := d.downloadAndInstall(ctx, resolvedVersion, cacheDir); err != nil {
		return "", fmt.Errorf("failed to download Deno: %w", err)
	}

	// Cleanup old versions
	if err := d.cleanupOldVersions(ctx, cacheDir); err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to cleanup old Deno versions: %s", err.Error()))
	}

	return binaryPath, nil
}

// denoBinaryName returns the platform-specific binary name.
func denoBinaryName() string {
	if runtime.GOOS == "windows" {
		return "deno.exe"
	}
	return "deno"
}

// getCacheDir returns the cache directory for Deno binaries.
func (d *DenoDownloader) getCacheDir() (string, error) {
	cacheDir := filepath.Join(os.TempDir(), "deno-tf-bridge")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}
	return cacheDir, nil
}

// getLatestVersion fetches the latest stable release version from GitHub.
func (d *DenoDownloader) getLatestVersion(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIBase, denoRepo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add GitHub token if available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if release.TagName == "" {
		return "", fmt.Errorf("no tag_name in release response")
	}

	return release.TagName, nil
}

// downloadAndInstall downloads and installs a specific version of Deno.
func (d *DenoDownloader) downloadAndInstall(ctx context.Context, version string, cacheDir string) error {
	// Get platform-specific asset name
	assetName, err := d.getPlatformAsset()
	if err != nil {
		return err
	}

	// Create version directory
	versionDir := filepath.Join(cacheDir, version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	// Fetch release info to get download URLs
	releaseInfo, err := d.getReleaseInfo(ctx, version)
	if err != nil {
		return err
	}

	// Find the asset and extract checksum from API
	var assetURL, expectedChecksum string
	for _, asset := range releaseInfo.Assets {
		if asset.Name == assetName {
			assetURL = asset.BrowserDownloadURL
			// Extract SHA256 hash from digest (format: "sha256:hash")
			if after, ok := strings.CutPrefix(asset.Digest, "sha256:"); ok {
				expectedChecksum = after
			}
			break
		}
	}

	if assetURL == "" {
		return fmt.Errorf("asset %s not found in release %s", assetName, version)
	}
	if expectedChecksum == "" {
		return fmt.Errorf("checksum not provided by GitHub API for asset %s in release %s", assetName, version)
	}

	tflog.Info(ctx, fmt.Sprintf("Downloading asset: %s", assetURL))
	tflog.Info(ctx, fmt.Sprintf("Expected checksum from GitHub API: %s", expectedChecksum))

	// Download the binary archive
	archivePath := filepath.Join(versionDir, assetName)
	if err := d.downloadFile(ctx, assetURL, archivePath); err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}

	// Verify checksum
	if err := d.verifyChecksum(archivePath, expectedChecksum); err != nil {
		os.Remove(archivePath)
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	tflog.Info(ctx, "Checksum verified successfully")

	// Extract the archive
	binaryPath := filepath.Join(versionDir, denoBinaryName())
	if err := d.extractArchive(archivePath, binaryPath); err != nil {
		os.Remove(archivePath)
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Remove the archive after extraction
	os.Remove(archivePath)

	// Make the binary executable on Unix systems
	if runtime.GOOS != "windows" {
		if err := os.Chmod(binaryPath, 0755); err != nil {
			return fmt.Errorf("failed to make binary executable: %w", err)
		}
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully installed Deno %s to %s", version, binaryPath))

	return nil
}

// getPlatformAsset returns the asset name for the current platform.
func (d *DenoDownloader) getPlatformAsset() (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var platform string
	switch {
	case goos == "windows" && goarch == "amd64":
		platform = "x86_64-pc-windows-msvc"
	case goos == "linux" && goarch == "amd64":
		platform = "x86_64-unknown-linux-gnu"
	case goos == "darwin" && goarch == "amd64":
		platform = "x86_64-apple-darwin"
	case goos == "darwin" && goarch == "arm64":
		platform = "aarch64-apple-darwin"
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s - Deno does not provide pre-built binaries for this operating system and architecture combination", goos, goarch)
	}

	return fmt.Sprintf("deno-%s%s", platform, ".zip"), nil
}

// getReleaseInfo fetches release information from GitHub.
func (d *DenoDownloader) getReleaseInfo(ctx context.Context, version string) (*githubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/tags/%s", githubAPIBase, denoRepo, version)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add GitHub token if available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &release, nil
}

// downloadFile downloads a file from a URL.
func (d *DenoDownloader) downloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// verifyChecksum verifies the SHA256 checksum of a file.
func (d *DenoDownloader) verifyChecksum(filePath, expectedChecksum string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// extractArchive extracts the Deno binary from a zip or tar.gz archive.
func (d *DenoDownloader) extractArchive(archivePath, destPath string) error {
	if strings.HasSuffix(archivePath, ".zip") {
		return d.extractZip(archivePath, destPath)
	} else if strings.HasSuffix(archivePath, ".tar.gz") {
		return d.extractTarGz(archivePath, destPath)
	}
	return fmt.Errorf("unsupported archive format: %s", archivePath)
}

// extractZip extracts the deno binary from a zip file.
func (d *DenoDownloader) extractZip(zipPath, destPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	// Find the deno binary in the zip
	for _, f := range r.File {
		if f.Name == denoBinaryName() || f.Name == "deno" {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("failed to open file in zip: %w", err)
			}
			defer rc.Close()

			out, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create destination file: %w", err)
			}
			defer out.Close()

			if _, err := io.Copy(out, rc); err != nil {
				return fmt.Errorf("failed to extract file: %w", err)
			}

			return nil
		}
	}

	return fmt.Errorf("deno binary not found in zip archive")
}

// extractTarGz extracts the deno binary from a tar.gz file.
func (d *DenoDownloader) extractTarGz(tarGzPath, destPath string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to open tar.gz: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	// Find the deno binary in the tar
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		if header.Name == denoBinaryName() || header.Name == "deno" {
			out, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create destination file: %w", err)
			}
			defer out.Close()

			if _, err := io.Copy(out, tr); err != nil {
				return fmt.Errorf("failed to extract file: %w", err)
			}

			return nil
		}
	}

	return fmt.Errorf("deno binary not found in tar.gz archive")
}

// cleanupOldVersions removes old Deno versions, keeping only the newest 3.
func (d *DenoDownloader) cleanupOldVersions(ctx context.Context, cacheDir string) error {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	// Parse versions
	type versionInfo struct {
		path    string
		version *semver.Version
	}

	var versions []versionInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Try to parse as semantic version
		v, err := semver.NewVersion(entry.Name())
		if err != nil {
			tflog.Debug(ctx, fmt.Sprintf("Skipping non-semver directory: %s", entry.Name()))
			continue
		}

		versions = append(versions, versionInfo{
			path:    filepath.Join(cacheDir, entry.Name()),
			version: v,
		})
	}

	// If we have 3 or fewer versions, nothing to clean up
	if len(versions) <= maxVersionsToKeep {
		return nil
	}

	// Sort by version descending (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].version.GreaterThan(versions[j].version)
	})

	// Remove versions beyond the first 3
	for i := maxVersionsToKeep; i < len(versions); i++ {
		tflog.Info(ctx, fmt.Sprintf("Removing old Deno version: %s", versions[i].version.String()))
		if err := os.RemoveAll(versions[i].path); err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Failed to remove %s: %s", versions[i].path, err.Error()))
		}
	}

	return nil
}
