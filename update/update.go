package update

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"secshell/drawbox"
	"strings"
)

const (
	DefaultVersion = "1.0.0" // Default version if not specified
	// Update script URL
	UpdateScript = "https://raw.githubusercontent.com/KaliforniaGator/SecShell-Go/refs/heads/main/update.sh"
	// Version URL
	VersionURL = "https://raw.githubusercontent.com/KaliforniaGator/SecShell-Go/refs/heads/main/VERSION"
)

// checkForUpdates checks if there's a new version available
func CheckForUpdates(versionFile string) {
	// Check if version file exists first
	if _, err := os.Stat(versionFile); os.IsNotExist(err) {
		// Version file doesn't exist, create it with latest version from GitHub
		resp, err := http.Get(VersionURL)
		if err != nil {
			// If offline, ensure we're using the default version
			UpdateVersionFile(DefaultVersion, versionFile)
			return
		}
		defer resp.Body.Close()

		// Read the response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			UpdateVersionFile(DefaultVersion, versionFile)
			return
		}

		// Update the version file with the latest version
		latestVersion := strings.TrimSpace(string(body))
		if latestVersion != "" {
			UpdateVersionFile(latestVersion, versionFile)
		} else {
			UpdateVersionFile(DefaultVersion, versionFile)
		}
	}
	// If version file exists, don't update it as update.sh will handle that
}

// updateVersionFile updates the .ver file with the given version
func UpdateVersionFile(version string, versionFile string) {
	err := os.WriteFile(versionFile, []byte(version+"\n"), 0644)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to update version file: %s", err))
	}
}

// getCurrentVersion gets the current version from the .ver file
func GetCurrentVersion(versionFile string) string {
	content, err := os.ReadFile(versionFile)
	if err != nil {
		return DefaultVersion
	}
	return strings.TrimSpace(string(content))
}

func GetLatestVersion() string {
	resp, err := http.Get(VersionURL)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to check version: %s", err))
		return DefaultVersion
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to read version data: %s", err))
		return DefaultVersion
	}
	return strings.TrimSpace(string(body))
}

// Display the current version of the shell
func DisplayVersion(versionFile string) {
	version := GetCurrentVersion(versionFile)
	drawbox.RunDrawbox(fmt.Sprintf("SecShell Version: %s", version), "bold_white")
}

// Update the current version of the shell
func UpdateSecShell(isAdmin bool, versionFile string) {
	// Check if user is an admin
	if !isAdmin {
		drawbox.PrintError("Permission denied: Admin privileges required for updates.")
		return
	}

	// Check if version is up to date at the beginning
	localVersion, err := os.ReadFile(versionFile)
	if err != nil {
		// If local version file doesn't exist or can't be read, assume update is needed
		drawbox.PrintAlert("Local version information not found. Proceeding with update...")
	} else {
		// Fetch the latest version from GitHub
		resp, err := http.Get("https://raw.githubusercontent.com/KaliforniaGator/SecShell-Go/refs/heads/main/VERSION")
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to check version: %s", err))
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to read version data: %s", err))
			return
		}
		githubVersion := strings.TrimSpace(string(body))
		localVersionStr := strings.TrimSpace(string(localVersion))

		// Compare versions
		if localVersionStr == githubVersion {
			drawbox.RunDrawbox("You're already up to date!", "bold_green")
			return
		}
	}

	// Create a temporary file for the update script
	tmpFile, err := os.CreateTemp("", "secshell-update-*.sh")
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to create temporary file: %s", err))
		return
	}
	defer os.Remove(tmpFile.Name()) // Clean up temp file when done

	// Download the update script with progress
	drawbox.PrintAlert("Downloading update script...")

	// Initialize HTTP client and request
	client := &http.Client{}
	req, err := http.NewRequest("GET", UpdateScript, nil)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to create request: %s", err))
		return
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to download update script: %s", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		drawbox.PrintError(fmt.Sprintf("Failed to download update script. Status: %s", resp.Status))
		return
	}

	// Get content length for progress bar
	contentLength := resp.ContentLength
	if contentLength <= 0 {
		contentLength = 1000 // Default size if unknown
	}

	// Create progress tracking reader
	progressReader := &ProgressReader{
		Reader: resp.Body,
		Total:  contentLength,
		UpdateFunc: func(bytesRead, total int64) {
			percent := int(math.Min(float64(bytesRead)/float64(total)*100, 100))
			// Run drawbox progress command
			cmd := exec.Command("drawbox", "progress", fmt.Sprintf("%d", percent), "100", "50", "block_full", "block_light", "green")
			cmd.Stdout = os.Stdout
			cmd.Run()
		},
	}

	// Save the update script to the temporary file
	_, err = io.Copy(tmpFile, progressReader)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to save update script: %s", err))
		return
	}
	tmpFile.Close()

	// Make the script executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to make update script executable: %s", err))
		return
	}

	// Run the update script
	drawbox.PrintAlert("Running update script...")
	cmd := exec.Command(tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		drawbox.PrintError(fmt.Sprintf("Update failed: %s", err))
		return
	}

	drawbox.PrintAlert("Update completed successfully. Restart SecShell to use the new version.")
	CheckForUpdates(versionFile)
}

// ProgressReader is a wrapper around an io.Reader that reports progress
type ProgressReader struct {
	Reader     io.Reader
	Total      int64
	BytesRead  int64
	UpdateFunc func(bytesRead, total int64)
}

// Read implements the io.Reader interface
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.BytesRead += int64(n)

	// Call the update function with the current progress
	if pr.UpdateFunc != nil {
		pr.UpdateFunc(pr.BytesRead, pr.Total)
	}

	return n, err
}
