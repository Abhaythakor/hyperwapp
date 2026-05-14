package detect

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	wappalyzer "github.com/projectdiscovery/wappalyzergo"
	"github.com/Abhaythakor/hyperwapp/model"
)

var (
	// Fast regex for pruning non-essential noise for technology detection.
	base64Regex = regexp.MustCompile(`data:[^;]+;base64,[A-Za-z0-9+/=]{50,}`)
	svgRegex    = regexp.MustCompile(`(?i)<svg[^>]*>.*?</svg>`)
	pathRegex   = regexp.MustCompile(`(?i)d="[A-Z0-9\s,.]{100,}"`)
	commentRegex = regexp.MustCompile(`(?s)<!--.*?-->`)
	styleRegex   = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	// Detect script blocks for manual truncation
	scriptBlockRegex = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	whitespaceRegex  = regexp.MustCompile(`\s{2,}`)
)

// wappalyzerClient defines the interface for the Wappalyzer client.
type wappalyzerClient interface {
	Fingerprint(headers map[string][]string, data []byte) map[string]struct{}
}

// Engine defines the interface for a technology detection engine.
type Engine interface {
	Detect(headers map[string][]string, body []byte, sourceHint string) ([]model.Detection, error)
}

// WappalyzerEngine implements the Engine interface using wappalyzergo.
type WappalyzerEngine struct {
	client    wappalyzerClient
	bodyCache sync.Map // [32]byte -> map[string]struct{}
}

// NewWappalyzerEngine creates and initializes a new WappalyzerEngine.
func NewWappalyzerEngine() (*WappalyzerEngine, error) {
	wappalyzerClient, err := wappalyzer.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize wappalyzer: %w", err)
	}
	return &WappalyzerEngine{
		client: wappalyzerClient,
	}, nil
}

// Detect identifies technologies based on headers and body.
func (e *WappalyzerEngine) Detect(headers map[string][]string, body []byte, sourceHint string) ([]model.Detection, error) {
	// Stage 1: Always scan headers (fast)
	fingerprints := e.client.Fingerprint(headers, nil)

	// Stage 2: Handle body scan
	if sourceHint != model.SourceHeadersOnly && len(body) > 0 {
		// Optimization: Check if it's a binary file first (fast)
		if e.isBinaryResponse(headers, body) {
			return e.wrapDetections(fingerprints, sourceHint), nil
		}

		// Optimization 1: Semantic Pruning (Removes non-detectable bloat)
		scanBody := e.pruneBody(body)

		// Optimization 2: Body Caching
		var bodyFingerprints map[string]struct{}
		if len(scanBody) > 1024 {
			bodyHash := sha256.Sum256(scanBody)
			if cached, ok := e.bodyCache.Load(bodyHash); ok {
				// Cache hit
				bodyFingerprints = cached.(map[string]struct{})
			} else {
				// Cache miss: Scan body only (once!)
				bodyFingerprints = e.client.Fingerprint(nil, scanBody)
				e.bodyCache.Store(bodyHash, bodyFingerprints)
			}
		} else {
			// Small body: Scan directly
			bodyFingerprints = e.client.Fingerprint(nil, scanBody)
		}

		// Merge body fingerprints into combined results
		for k := range bodyFingerprints {
			fingerprints[k] = struct{}{}
		}
	}

	return e.wrapDetections(fingerprints, sourceHint), nil
}

// isBinaryResponse checks if the response is a non-textual format that should skip body scanning.
func (e *WappalyzerEngine) isBinaryResponse(headers map[string][]string, body []byte) bool {
	// 1. Check Content-Type header
	ct := ""
	if v, ok := headers["Content-Type"]; ok && len(v) > 0 {
		ct = strings.ToLower(v[0])
	} else if v, ok := headers["content-type"]; ok && len(v) > 0 {
		ct = strings.ToLower(v[0])
	}

	if ct != "" {
		// Skip common binary formats
		binaryPrefixes := []string{
			"image/", "audio/", "video/", "font/",
			"application/octet-stream", "application/pdf",
			"application/zip", "application/x-gzip", "application/x-rar",
		}
		for _, prefix := range binaryPrefixes {
			if strings.HasPrefix(ct, prefix) {
				return true
			}
		}
	}

	// 2. Heuristic: Check for null bytes in the first 512 bytes of the body
	// (Standard way to detect binary files if headers are missing)
	checkLen := len(body)
	if checkLen > 512 {
		checkLen = 512
	}
	if bytes.IndexByte(body[:checkLen], 0) != -1 {
		return true
	}

	return false
}

// pruneBody removes massive Base64 blobs, SVGs, comments, and styles to speed up regex scanning.
func (e *WappalyzerEngine) pruneBody(body []byte) []byte {
	// Avoid pruning small bodies
	if len(body) < 2048 {
		return body
	}

	// 1. Remove comments
	pruned := commentRegex.ReplaceAll(body, []byte(""))
	// 2. Remove styles
	pruned = styleRegex.ReplaceAll(pruned, []byte(""))
	// 3. Remove SVGs
	pruned = svgRegex.ReplaceAll(pruned, []byte("[svg]"))
	pruned = pathRegex.ReplaceAll(pruned, []byte(`d=""`))
	
	// 4. Truncate huge scripts manually (keep first 5KB)
	pruned = scriptBlockRegex.ReplaceAllFunc(pruned, func(script []byte) []byte {
		if len(script) > 5000 {
			openTagEnd := bytes.IndexByte(script, '>')
			if openTagEnd != -1 && openTagEnd < 1000 {
				newScript := make([]byte, 0, 5100)
				newScript = append(newScript, script[:openTagEnd+1]...)
				newScript = append(newScript, []byte("...[truncated]")...)
				newScript = append(newScript, []byte("</script>")...)
				return newScript
			}
			return []byte("<script>[truncated]</script>")
		}
		return script
	})

	// 5. Remove base64 strings
	pruned = base64Regex.ReplaceAll(pruned, []byte("[b64]"))
	// 6. Collapse whitespace
	pruned = whitespaceRegex.ReplaceAll(pruned, []byte(" "))
	
	return pruned
}

func (e *WappalyzerEngine) wrapDetections(fingerprints map[string]struct{}, sourceHint string) []model.Detection {
	detections := make([]model.Detection, 0, len(fingerprints))
	now := time.Now().UTC()

	source := model.SourceWappalyzer
	if sourceHint == model.SourceHeadersOnly {
		source = model.SourceHeadersOnly
	} else if sourceHint == model.SourceBodyOnly {
		source = model.SourceBodyOnly
	}

	for tech := range fingerprints {
		detections = append(detections, model.Detection{
			Technology: tech,
			Source:     source,
			Path:       "fingerprint",
			Evidence:   "wappalyzergo",
			Confidence: "high",
			Timestamp:  now,
		})
	}
	return detections
}
