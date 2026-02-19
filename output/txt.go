package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sort" // Added sort package import
	"strings"
	"sync"

	"hyperwapp/aggregate"
	"hyperwapp/model"
)

// TXTWriter implements the Writer interface for plain text output.
type TXTWriter struct {
	filePath string
	mu       sync.Mutex // To ensure safe concurrent writes if needed
	file     *os.File
	mode     string
	tempFile *os.File
}

// NewTXTWriter creates a new TXTWriter.
func NewTXTWriter(filePath string) (*TXTWriter, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create TXT output file %s: %w", filePath, err)
	}
	return &TXTWriter{filePath: filePath, file: file, mode: "all"}, nil
}

// Write outputs detections for individual targets to the TXT file or buffers them for domain mode.
func (w *TXTWriter) Write(detections []model.Detection) error {
	if len(detections) == 0 {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.mode == "domain" {
		if w.tempFile == nil {
			var err error
			w.tempFile, err = os.CreateTemp("", "HyperWapp-txt-*.jsonl")
			if err != nil {
				return err
			}
		}
		encoder := json.NewEncoder(w.tempFile)
		for _, d := range detections {
			if err := encoder.Encode(d); err != nil {
				return err
			}
		}
		return nil
	}

	// Group by URL
	targets := make(map[string][]model.Detection)
	for _, d := range detections {
		key := d.URL
		if key == "" {
			key = d.Domain
		}
		targets[key] = append(targets[key], d)
	}

	for target, targetDetections := range targets {
		if len(targetDetections) == 0 {
			continue
		}
		domain := targetDetections[0].Domain

		builder := strings.Builder{}
		builder.WriteString(fmt.Sprintf("URL: %s\n", target))
		builder.WriteString(fmt.Sprintf("Domain: %s\n", domain))
		builder.WriteString("  Technologies:\n")
		for _, d := range targetDetections {
			builder.WriteString(fmt.Sprintf("    - %s (Source: %s, Confidence: %s)\n", d.Technology, d.Source, d.Confidence))
		}
		builder.WriteString("\n")

		if _, err := w.file.WriteString(builder.String()); err != nil {
			return fmt.Errorf("failed to write to TXT file: %w", err)
		}
	}
	
	return nil
}

// SetMode updates the output mode.
func (w *TXTWriter) SetMode(mode string) {
	w.mode = mode
}

// WriteAggregated outputs aggregated detections to the TXT file.
func (w *TXTWriter) WriteAggregated(aggregated []aggregate.AggregatedDomain) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, agg := range aggregated {
		if len(agg.Detections) == 0 {
			continue
		}

		builder := strings.Builder{}
		builder.WriteString(fmt.Sprintf("Domain: %s\n", agg.Domain))
		builder.WriteString(fmt.Sprintf("  URLs Scanned: %d\n", len(agg.URLs)))
		if len(agg.URLs) < 50 {
			for _, u := range agg.URLs {
				builder.WriteString(fmt.Sprintf("    - %s\n", u))
			}
		} else {
			builder.WriteString(fmt.Sprintf("    - (and %d more URLs...)\n", len(agg.URLs)-1))
		}
		builder.WriteString("  Technologies:\n")
		uniqueTechs := make(map[string]struct{})
		for _, d := range agg.Detections {
			uniqueTechs[d.Technology] = struct{}{}
		}
		var sortedTechs []string
		for tech := range uniqueTechs {
			sortedTechs = append(sortedTechs, tech)
		}
		sort.Strings(sortedTechs)

		for _, tech := range sortedTechs {
			builder.WriteString(fmt.Sprintf("    - %s\n", tech))
		}
		builder.WriteString("\n")

		if _, err := w.file.WriteString(builder.String()); err != nil {
			return fmt.Errorf("failed to write to TXT file: %w", err)
		}
	}
	return nil
}

// Close closes the underlying file and handles domain aggregation if needed.
func (w *TXTWriter) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.mode == "domain" && w.tempFile != nil {
		defer os.Remove(w.tempFile.Name())
		defer w.tempFile.Close()

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
				}
			}
			found := false
			for _, u := range domainMap[d.Domain].URLs {
				if u == d.URL {
					found = true
					break
				}
			}
			if !found && d.URL != "" {
				domainMap[d.Domain].URLs = append(domainMap[d.Domain].URLs, d.URL)
			}
			domainMap[d.Domain].Detections = append(domainMap[d.Domain].Detections, d)
		}

		var sortedDomains []string
		for d := range domainMap {
			sortedDomains = append(sortedDomains, d)
		}
		sort.Strings(sortedDomains)

		for _, d := range sortedDomains {
			// Temporarily unlock to call WriteAggregated which also locks (or just inline it)
			// For safety, let's inline it here since we already have the lock.
			agg := domainMap[d]
			builder := strings.Builder{}
			builder.WriteString(fmt.Sprintf("Domain: %s\n", agg.Domain))
			builder.WriteString(fmt.Sprintf("  URLs Scanned: %d\n", len(agg.URLs)))
			if len(agg.URLs) < 50 {
				for _, u := range agg.URLs {
					builder.WriteString(fmt.Sprintf("    - %s\n", u))
				}
			} else {
				builder.WriteString(fmt.Sprintf("    - (and %d more URLs...)\n", len(agg.URLs)-1))
			}
			builder.WriteString("  Technologies:\n")
			uniqueTechs := make(map[string]struct{})
			for _, det := range agg.Detections {
				uniqueTechs[det.Technology] = struct{}{}
			}
			var sortedTechs []string
			for tech := range uniqueTechs {
				sortedTechs = append(sortedTechs, tech)
			}
			sort.Strings(sortedTechs)
			for _, tech := range sortedTechs {
				builder.WriteString(fmt.Sprintf("    - %s\n", tech))
			}
			builder.WriteString("\n")
			_, _ = w.file.WriteString(builder.String())
		}
	}

	if w.file != nil {
		w.file.Close()
	}
}
