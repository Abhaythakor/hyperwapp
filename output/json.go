package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Abhaythakor/hyperwapp/aggregate" // Added aggregate package import
	"github.com/Abhaythakor/hyperwapp/model"
)

// JSONWriter implements the Writer interface for JSON output with disk-backed streaming.
type JSONWriter struct {
	filePath  string
	inputType string
	version   string
	tempFile  *os.File
	written   bool   // To prevent double writing in Close
	Mode      string // all | domain
	mu        sync.Mutex
}

// NewJSONWriter creates a new JSONWriter.
func NewJSONWriter(filePath string, inputType string, version string) (*JSONWriter, error) {
	// Create a temporary file to store raw detections (JSONL format)
	tempFile, err := os.CreateTemp("", "HyperWapp-*.jsonl")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file for JSON results: %w", err)
	}

	return &JSONWriter{
		filePath:  filePath,
		inputType: inputType,
		version:   version,
		tempFile:  tempFile,
		Mode:      "all", // Default mode
	}, nil
}

// Write streams incoming detections to the temporary file.
func (w *JSONWriter) Write(detections []model.Detection) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	encoder := json.NewEncoder(w.tempFile)
	for _, d := range detections {
		if err := encoder.Encode(d); err != nil {
			return fmt.Errorf("failed to encode detection to temp file: %w", err)
		}
	}
	return nil
}

// SetMode sets the output mode (all | domain).
func (w *JSONWriter) SetMode(mode string) {
	w.Mode = mode
}

// WriteAggregated is NOT disk-backed in this simple implementation because aggregation
// naturally requires buffering or a multi-pass approach.
// However, we'll make it work by reading from the temp file if needed.
func (w *JSONWriter) WriteAggregated(aggregated []aggregate.AggregatedDomain) error {
	w.written = true
	outputData := struct {
		Meta    model.Meta                   `json:"meta"`
		Results []aggregate.AggregatedDomain `json:"results"`
	}{
		Meta: model.Meta{
			Tool:        "HyperWapp",
			Version:     w.version,
			GeneratedAt: time.Now().UTC(),
			Mode:        w.Mode,
			InputType:   w.inputType,
		},
		Results: aggregated,
	}

	file, err := os.Create(w.filePath)
	if err != nil {
		return fmt.Errorf("failed to create JSON output file %s: %w", w.filePath, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(outputData)
}

// Close finalize the JSON output by reading from the temp file and writing the structured format.
func (w *JSONWriter) Close() {
	if w.written {
		os.Remove(w.tempFile.Name())
		return
	}
	w.written = true
	defer os.Remove(w.tempFile.Name())

	finalFile, err := os.Create(w.filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating final JSON file: %v\n", err)
		return
	}
	defer finalFile.Close()

	if w.Mode == "domain" {
		w.finalizeDomainMode(finalFile)
	} else {
		w.finalizeAllMode(finalFile)
	}
	w.tempFile.Close()
}

func (w *JSONWriter) finalizeAllMode(finalFile *os.File) {
	// Write Meta
	meta := model.Meta{
		Tool:        "HyperWapp",
		Version:     w.version,
		GeneratedAt: time.Now().UTC(),
		Mode:        "all",
		InputType:   w.inputType,
	}

	fmt.Fprintf(finalFile, "{\n  \"meta\": ")
	metaBytes, _ := json.MarshalIndent(meta, "  ", "  ")
	finalFile.Write(metaBytes)
	fmt.Fprintf(finalFile, ",\n  \"results\": [\n")

	w.tempFile.Seek(0, 0)
	decoder := json.NewDecoder(w.tempFile)
	first := true
	currentURL := ""
	var currentDetections []model.Detection

	writeResult := func(url string, detections []model.Detection) {
		if !first {
			fmt.Fprintf(finalFile, ",\n")
		}
		first = false
		result := struct {
			Domain     string            `json:"domain"`
			URL        string            `json:"url"`
			Detections []model.Detection `json:"detections"`
		}{
			Domain:     detections[0].Domain,
			URL:        url,
			Detections: detections,
		}
		resBytes, _ := json.MarshalIndent(result, "    ", "  ")
		finalFile.Write(resBytes)
	}

	for {
		var d model.Detection
		if err := decoder.Decode(&d); err != nil {
			break
		}
		if d.URL != currentURL {
			if currentURL != "" {
				writeResult(currentURL, currentDetections)
			}
			currentURL = d.URL
			currentDetections = []model.Detection{d}
		} else {
			currentDetections = append(currentDetections, d)
		}
	}
	if currentURL != "" {
		writeResult(currentURL, currentDetections)
	}
	fmt.Fprintf(finalFile, "\n  ]\n}\n")
}

func (w *JSONWriter) finalizeDomainMode(finalFile *os.File) {
	// Aggregation requires grouping. To save RAM, we use a map of Slices,
	// but we only store the detections, not the full objects if possible.
	// For 10M targets, this is the most RAM-intensive part.

	domainMap := make(map[string]*aggregate.AggregatedDomain)

	w.tempFile.Seek(0, 0)
	decoder := json.NewDecoder(w.tempFile)
	for {
		var d model.Detection
		if err := decoder.Decode(&d); err != nil {
			break
		}

		if _, ok := domainMap[d.Domain]; !ok {
			domainMap[d.Domain] = &aggregate.AggregatedDomain{
				Domain: d.Domain,
				URLs:   []string{},
			}
		}

		// Add unique URL
		foundURL := false
		for _, u := range domainMap[d.Domain].URLs {
			if u == d.URL {
				foundURL = true
				break
			}
		}
		if !foundURL && d.URL != "" {
			domainMap[d.Domain].URLs = append(domainMap[d.Domain].URLs, d.URL)
		}

		domainMap[d.Domain].Detections = append(domainMap[d.Domain].Detections, d)
	}

	// Write Meta
	meta := model.Meta{
		Tool:        "HyperWapp",
		Version:     w.version,
		GeneratedAt: time.Now().UTC(),
		Mode:        "domain",
		InputType:   w.inputType,
	}

	fmt.Fprintf(finalFile, "{\n  \"meta\": ")
	metaBytes, _ := json.MarshalIndent(meta, "  ", "  ")
	finalFile.Write(metaBytes)
	fmt.Fprintf(finalFile, ",\n  \"results\": [\n")

	first := true
	for _, agg := range domainMap {
		if !first {
			fmt.Fprintf(finalFile, ",\n")
		}
		first = false
		resBytes, _ := json.MarshalIndent(agg, "    ", "  ")
		finalFile.Write(resBytes)
	}
	fmt.Fprintf(finalFile, "\n  ]\n}\n")
}
