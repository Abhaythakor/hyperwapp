package detect

import (
	"regexp"
	"strings"
)

var (
	// Manual overrides for technologies that don't follow the slug pattern
	nucleiManualMap = map[string]string{
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
	}

	nonAlphaRegex = regexp.MustCompile(`[^a-z0-9]+`)
)

// MapToNucleiTag converts a Wappalyzer technology name to a Nuclei-compatible tag.
func MapToNucleiTag(tech string) string {
	// 1. Check manual map first
	if tag, ok := nucleiManualMap[tech]; ok {
		return tag
	}

	// 2. Smart Slugifier:
	// "WordPress" -> "wordpress"
	// "PHP 7.4" -> "php"
	// "Google Analytics" -> "google-analytics"
	
	tag := strings.ToLower(tech)
	
	// Remove version numbers (e.g., "php 7.4" -> "php ")
	versionIdx := strings.IndexAny(tag, "0123456789")
	if versionIdx > 0 {
		tag = tag[:versionIdx]
	}

	// Clean special characters and trim
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
