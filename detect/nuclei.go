package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/Abhaythakor/hyperwapp/util"
)

var (
	// Manual overrides and loaded JSON map
	nucleiMap   = make(map[string]string)
	mapMutex    sync.RWMutex
	mapLoaded   bool
	nonAlphaRegex = regexp.MustCompile(`[^a-z0-9]+`)
)

// LoadNucleiMap loads the mapping from a JSON file.
func LoadNucleiMap() {
	mapMutex.Lock()
	defer mapMutex.Unlock()

	if mapLoaded {
		return
	}

	// Try to find nuclei-map.json in the executable directory or current directory
	exePath, _ := os.Executable()
	paths := []string{
		filepath.Join(filepath.Dir(exePath), "nuclei-map.json"),
		"nuclei-map.json",
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			err = json.Unmarshal(data, &nucleiMap)
			if err == nil {
				util.Debug("Loaded Nuclei mapping from %s", p)
				mapLoaded = true
				return
			}
		}
	}

	// Fallback hardcoded defaults if file not found
	nucleiMap = map[string]string{
		"Amazon Web Services": "aws",
		"Google Cloud":        "gcp",
		"Microsoft Azure":     "azure",
		"F5 BigIP":            "f5",
		"Google Cloud Storage": "gcs",
		"Apache HTTP Server":  "apache",
		"Nginx HTTP Server":   "nginx",
		"Ruby on Rails":       "rails",
		"Node.js":             "nodejs",
		"Vue.js":              "vue",
		"React.js":            "react",
		"WordPress":           "wordpress",
	}
	mapLoaded = true
}

// MapToNucleiTag converts a Wappalyzer technology name to a Nuclei-compatible tag.
func MapToNucleiTag(tech string) string {
	if !mapLoaded {
		LoadNucleiMap()
	}

	mapMutex.RLock()
	tag, ok := nucleiMap[tech]
	mapMutex.RUnlock()

	if ok {
		return tag
	}

	// Smart Slugifier (Fallback)
	tag = strings.ToLower(tech)
	versionIdx := strings.IndexAny(tag, "0123456789")
	if versionIdx > 0 {
		tag = tag[:versionIdx]
	}
	tag = strings.TrimSpace(tag)
	tag = nonAlphaRegex.ReplaceAllString(tag, "-")
	tag = strings.Trim(tag, "-")

	return tag
}

// MapToNucleiTags converts a slice of technologies to a unique slice of Nuclei tags.
func MapToNucleiTags(techs []string) []string {
	tagMap := make(map[string]struct{})
	var tags []string

	for _, tech := range techs {
		tag := MapToNucleiTag(tech)
		if tag != "" {
			if _, exists := tagMap[tag]; !exists {
				tagMap[tag] = struct{}{}
				tags = append(tags, tag)
			}
		}
	}
	return tags
}
