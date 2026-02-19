package output

import (
	"hyperwapp/aggregate" // Added aggregate package import
	"hyperwapp/model"
)

// Writer defines the interface for outputting detection results.
type Writer interface {
	// Write outputs a batch of detections.
	Write(detections []model.Detection) error
	// WriteAggregated outputs detections grouped by domain.
	WriteAggregated(aggregated []aggregate.AggregatedDomain) error
	// SetMode sets the output mode (all | domain).
	SetMode(mode string)
	// Close finalizes and closes the writer.
	Close()
}
