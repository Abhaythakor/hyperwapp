package model

import (
	"fmt"
	"sync"
	"time"
)

// LinePool is a global pool for reusable byte buffers for raw data ingestion.
var LinePool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 1024*1024) // 1MB initial capacity
	},
}

// OfflineInput represents the normalized input from offline sources.
type OfflineInput struct {
	Domain   string
	URL      string
	Headers  map[string][]string
	Body     []byte
	Path     string // Source file path for fast resume
	Skipped  bool   // True if this item was already processed (resume mode)
	RawJSON  []byte // Raw JSON line for parallel parsing
	RawRegex []byte // Raw Regex record for parallel parsing
}

// Reset clears the fields of an OfflineInput for reuse.
func (i *OfflineInput) Reset() {
	i.Domain = ""
	i.URL = ""
	i.Path = ""
	i.Skipped = false
	i.Body = nil

	// Note: We don't Reset RawJSON/RawRegex here because they might point to pooled buffers
	// that need to be returned to LinePool explicitly before Reset if they were used.
	// However, for safety, we clear the references.
	i.RawJSON = nil
	i.RawRegex = nil

	// Clear the headers map without reallocating
	if i.Headers != nil {
		for k := range i.Headers {
			delete(i.Headers, k)
		}
	}
}

// OfflineInputPool is a global pool for reusing OfflineInput objects to reduce GC pressure
var OfflineInputPool = sync.Pool{
	New: func() interface{} {
		return &OfflineInput{
			Headers: make(map[string][]string),
		}
	},
}

// Detection represents a single identified technology on a target.
type Detection struct {
	Domain     string    `json:"domain" csv:"domain"`         // example.com
	URL        string    `json:"url" csv:"url"`               // https://example.com
	Technology string    `json:"technology" csv:"technology"` // React, Cloudflare, Apache
	NucleiTags []string  `json:"nuclei_tags,omitempty" csv:"nuclei_tags,omitempty"` // wordpress, php, etc
	Source     string    `json:"source" csv:"source"`         // wappalyzer
	Path       string    `json:"path" csv:"path"`             // fingerprint
	Evidence   string    `json:"evidence" csv:"evidence"`     // wappalyzergo
	Confidence string    `json:"confidence" csv:"confidence"` // high
	Timestamp  time.Time `json:"timestamp" csv:"timestamp"`   // RFC3339
}

// Validate performs schema validation on an OfflineInput struct.
func (i *OfflineInput) Validate() error {
	if i.Domain == "" {
		return fmt.Errorf("OfflineInput: Domain cannot be empty")
	}
	if i.Headers == nil {
		return fmt.Errorf("OfflineInput: Headers map must not be nil")
	}
	return nil
}
