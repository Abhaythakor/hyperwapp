package detect

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"hyperwapp/util"
)

const (
	fingerprintsURL = "https://raw.githubusercontent.com/projectdiscovery/wappalyzergo/master/fingerprints_data.json"
)

// GetFingerprintsPath returns the local path where wappalyzergo looks for fingerprints.
func GetFingerprintsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Default path used by wappalyzergo
	return filepath.Join(home, ".config", "wappalyzergo", "fingerprints.json"), nil
}

// UpdateFingerprints downloads the latest fingerprints from ProjectDiscovery.
func UpdateFingerprints() error {
	util.Info("Updating Wappalyzer fingerprints...")

	path, err := GetFingerprintsPath()
	if err != nil {
		return fmt.Errorf("could not determine fingerprints path: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	// Download file
	resp, err := http.Get(fingerprintsURL)
	if err != nil {
		return fmt.Errorf("failed to download fingerprints: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", path, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save fingerprints: %w", err)
	}

	util.Info("Fingerprints updated successfully to %s", path)
	return nil
}

// GetFingerprintsInfo returns information about the local fingerprints file.
func GetFingerprintsInfo() string {
	path, err := GetFingerprintsPath()
	if err != nil {
		return "Unknown"
	}

	info, err := os.Stat(path)
	if err != nil {
		return "Not found (using embedded defaults)"
	}

	return fmt.Sprintf("Last updated: %s", info.ModTime().Format("2006-01-02 15:04:05"))
}
