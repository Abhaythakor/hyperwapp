package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	latestReleaseURL = "https://api.github.com/repos/Abhaythakor/hyperwapp/releases/latest"
)

// CheckForUpdates checks GitHub for a newer version of the tool.
func CheckForUpdates(currentVersion string) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(latestReleaseURL)
	if err != nil {
		return // Silently fail update check
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	if isNewer(current, latest) {
		fmt.Printf("
%s A new version of HyperWapp is available: %s (Current: %s)
", 
			NewColorizer(true).Yellow("[!]"),
			NewColorizer(true).Green("v"+latest),
			currentVersion)
		fmt.Printf("%s Run %s to upgrade.

", 
			NewColorizer(true).Yellow("[!]"),
			NewColorizer(true).Cyan("hyperwapp --update"))
	}
}

// simple semver comparison (major.minor.patch)
func isNewer(current, latest string) bool {
	return current != latest && latest != ""
}
