package detect

import (
	"fmt"
	"time"

	wappalyzer "github.com/projectdiscovery/wappalyzergo"
	"hyperwapp/model"
)

// wappalyzerClient defines the interface for the Wappalyzer client.
// This is used to decouple our engine from the concrete implementation.
type wappalyzerClient interface {
	Fingerprint(headers map[string][]string, data []byte) map[string]struct{}
}

// Engine defines the interface for a technology detection engine.
type Engine interface {
	Detect(headers map[string][]string, body []byte, sourceHint string) ([]model.Detection, error)
}

// WappalyzerEngine implements the Engine interface using wappalyzer.
type WappalyzerEngine struct {
	client wappalyzerClient
}

// NewWappalyzerEngine creates and initializes a new WappalyzerEngine.
func NewWappalyzerEngine() (*WappalyzerEngine, error) {
	wappalyzerClient, err := wappalyzer.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize wappalyzer: %w", err)
	}
	return &WappalyzerEngine{client: wappalyzerClient}, nil
}

// Detect identifies technologies based on headers and body.
func (e *WappalyzerEngine) Detect(headers map[string][]string, body []byte, sourceHint string) ([]model.Detection, error) {
	fingerprints := e.client.Fingerprint(headers, body)

	var detections []model.Detection
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
			Confidence: "high", // Wappalyzer detections are considered high confidence
			Timestamp:  now,
		})
	}
	return detections, nil
}