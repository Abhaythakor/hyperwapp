package model

import (
	"fmt" // Added import
	"time"
)

type Detection struct {
	Domain     string    `json:"domain" csv:"domain"`         // example.com
	URL        string    `json:"url" csv:"url"`               // https://example.com
	Technology string    `json:"technology" csv:"technology"` // React, Cloudflare, Apache
	Source     string    `json:"source" csv:"source"`         // wappalyzer
	Path       string    `json:"path" csv:"path"`             // fingerprint
	Evidence   string    `json:"evidence" csv:"evidence"`     // wappalyzergo
	Confidence string    `json:"confidence" csv:"confidence"` // high
	Timestamp  time.Time `json:"timestamp" csv:"timestamp"`   // RFC3339
}

// OfflineInput represents the normalized input from offline sources
type OfflineInput struct {
	Domain  string
	URL     string
	Headers map[string][]string
	Body    []byte
}

// Validate performs schema validation on an OfflineInput struct.
func (i OfflineInput) Validate() error {
	if i.Domain == "" {
		return fmt.Errorf("OfflineInput: Domain cannot be empty")
	}
	if i.Headers == nil {
		return fmt.Errorf("OfflineInput: Headers map must not be nil")
	}
	// Body can be empty
	// URL can be empty (e.g., in domain-only mode)
	return nil
}
