package output

import (
	"hyperwapp/aggregate" // Added aggregate package import
	"hyperwapp/model"
)

// Writer defines the interface for outputting detection results.
type Writer interface {
	Write(detections []model.Detection) error // For streaming individual detections or small batches
	WriteAggregated(aggregated []aggregate.AggregatedDomain) error
	SetMode(mode string)
	Close()
}
