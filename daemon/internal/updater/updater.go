package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	githubOwner = "akiyax"
	githubRepo  = "claudepilot"
	checkTimeout = 10 * time.Second
)

// ReleaseInfo represents a GitHub Release.
type ReleaseInfo struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"` // Release notes
}

// Updater checks for and downloads daemon updates.
type Updater struct {
	currentVersion string
	httpClient     *http.Client
}

// NewUpdater creates a new updater.
func NewUpdater(currentVersion string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		httpClient: &http.Client{
			Timeout: checkTimeout,
		},
	}
}

// CheckUpdate checks GitHub Releases for a newer version.
// Returns the release info if an update is available, nil if up-to-date.
func (u *Updater) CheckUpdate() (*ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)

	resp, err := u.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parsing release info: %w", err)
	}

	// Compare versions (simple string comparison for now)
	if release.TagName == "" || release.TagName == u.currentVersion {
		return nil, nil // Up to date
	}

	return &release, nil
}

// DownloadUpdate downloads the latest binary for the current platform.
// Returns the path to the downloaded file.
func (u *Updater) DownloadUpdate(version string) (string, error) {
	// Determine platform-specific binary name
	binaryName := fmt.Sprintf("claudepilot-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		githubOwner, githubRepo, version, binaryName)

	resp, err := u.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("downloading update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "claudepilot-update-*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Download to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("writing update: %w", err)
	}
	tmpFile.Close()

	// Make executable
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("chmod: %w", err)
	}

	return tmpPath, nil
}

// ApplyUpdate replaces the current binary with the downloaded update.
func (u *Updater) ApplyUpdate(updatePath string) error {
	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting current exe path: %w", err)
	}

	// Resolve symlinks
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("resolving exe path: %w", err)
	}

	// Backup current binary
	backupPath := currentExe + ".bak"
	if err := os.Rename(currentExe, backupPath); err != nil {
		return fmt.Errorf("backing up current binary: %w", err)
	}

	// Move update into place
	if err := os.Rename(updatePath, currentExe); err != nil {
		// Rollback on failure
		os.Rename(backupPath, currentExe)
		return fmt.Errorf("replacing binary: %w", err)
	}

	// Clean up backup
	os.Remove(backupPath)

	return nil
}

// SelfUpdate performs a complete self-update: check → download → apply.
func (u *Updater) SelfUpdate() error {
	release, err := u.CheckUpdate()
	if err != nil {
		return fmt.Errorf("check update: %w", err)
	}
	if release == nil {
		return fmt.Errorf("already up to date")
	}

	updatePath, err := u.DownloadUpdate(release.TagName)
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}

	if err := u.ApplyUpdate(updatePath); err != nil {
		return fmt.Errorf("apply update: %w", err)
	}

	return nil
}
