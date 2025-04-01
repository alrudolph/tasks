package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const repoOwner = "alrudolph"
const repoName = "tasks"

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func getLatestRelease() (string, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to fetch latest release: %s", resp.Status)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", err
	}

	if len(release.Assets) == 0 {
		return "", "", fmt.Errorf("no assets found in the latest release")
	}

	return release.TagName, release.Assets[0].BrowserDownloadURL, nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func replaceBinary(newBinaryPath string) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	// Rename old binary (optional)
	backupPath := exePath + ".old"
	err = os.Rename(exePath, backupPath)

	if err != nil {
		return err
	}

	// Move new binary in place
	err = os.Rename(newBinaryPath, exePath)
	if err != nil {
		return err
	}

	// Set executable permissions
	err = os.Chmod(exePath, 0755)
	if err != nil {
		return err
	}

	fmt.Println("Upgrade successful!")
	return nil
}

func upgrade() error {
	latestVersion, downloadURL, err := getLatestRelease()
	if err != nil {
		return err
	}

	fmt.Println("Latest version:", latestVersion)

	// Determine temp path for download
	tmpFile := filepath.Join(os.TempDir(), "program-new")

	err = downloadFile(downloadURL, tmpFile)
	if err != nil {
		return err
	}

	err = replaceBinary(tmpFile)
	if err != nil {
		return err
	}

	fmt.Println("Upgrade complete! Restart the program.")
	return nil
}

// // TODO:
// // * Make sure the release assets on GitHub match the OS/architecture (e.g., program-linux-amd64, program-mac-arm64)
// // * Handle cases where the user runs the program without admin/sudo (they may need to install in ~/.local/bin instead of /usr/local/bin)
// // * Optionally add a checksum validation step for security
