package output

import (
	"encoding/csv"
	"os"
	"time"

	"hyperwapp/aggregate" // Added aggregate package import
	"hyperwapp/model"
)

// CSVWriter implements the Writer interface for CSV output.
type CSVWriter struct {
	writer *csv.Writer
	mode   string
}

// NewCSVWriter creates a new CSVWriter.
func NewCSVWriter(filePath string, appendMode bool) (*CSVWriter, error) {
	flags := os.O_CREATE | os.O_WRONLY
	isNew := true
	if appendMode {
		if _, err := os.Stat(filePath); err == nil {
			isNew = false
		}
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(filePath, flags, 0644)
	if err != nil {
		return nil, err
	}

	w := csv.NewWriter(file)
	// Write header only if new file
	if isNew {
		header := []string{"domain", "url", "technology", "source", "path", "evidence", "confidence", "timestamp"}
		if err := w.Write(header); err != nil {
			return nil, err
		}
	}

	return &CSVWriter{writer: w, mode: "all"}, nil
}

// Write outputs detections for individual targets to the CSV file.
func (w *CSVWriter) Write(detections []model.Detection) error {
	for _, d := range detections {
		url := d.URL
		if w.mode == "domain" {
			url = ""
		}
		record := []string{
			d.Domain,
			url,
			d.Technology,
			d.Source,
			d.Path,
			d.Evidence,
			d.Confidence,
			d.Timestamp.Format(time.RFC3339),
		}
		if err := w.writer.Write(record); err != nil {
			return err
		}
	}
	w.writer.Flush() // Flush after each batch to ensure data is written
	return w.writer.Error()
}

// SetMode updates the output mode.
func (w *CSVWriter) SetMode(mode string) {
	w.mode = mode
}

// WriteAggregated outputs aggregated detections to the CSV file.
func (w *CSVWriter) WriteAggregated(aggregated []aggregate.AggregatedDomain) error {
	for _, agg := range aggregated {
		for _, d := range agg.Detections {
			record := []string{
				agg.Domain, // Use aggregated domain
				d.URL,      // Keep original URL from detection
				d.Technology,
				d.Source,
				d.Path,
				d.Evidence,
				d.Confidence,
				d.Timestamp.Format(time.RFC3339),
			}
			if err := w.writer.Write(record); err != nil {
				return err
			}
		}
	}
	w.writer.Flush()
	return w.writer.Error()
}

// Close flushes any buffered data and closes the underlying file.
func (w *CSVWriter) Close() {
	w.writer.Flush()
}
