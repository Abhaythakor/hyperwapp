package output

import (
	"encoding/json"
	"os"
	"sync"

	"hyperwapp/aggregate"
	"hyperwapp/model"
)

// JSONLWriter implements the Writer interface for JSON Lines output.
// It is highly efficient and appendable.
type JSONLWriter struct {
	file *os.File
	mu   sync.Mutex
	mode string
}

// NewJSONLWriter creates a new JSONLWriter.
func NewJSONLWriter(filePath string, appendMode bool) (*JSONLWriter, error) {
	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(filePath, flags, 0644)
	if err != nil {
		return nil, err
	}

	return &JSONLWriter{file: file, mode: "all"}, nil
}

func (w *JSONLWriter) Write(detections []model.Detection) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	encoder := json.NewEncoder(w.file)
	for _, d := range detections {
		if err := encoder.Encode(d); err != nil {
			return err
		}
	}
	return nil
}

func (w *JSONLWriter) SetMode(mode string) {
	w.mode = mode
}

func (w *JSONLWriter) WriteAggregated(aggregated []aggregate.AggregatedDomain) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	encoder := json.NewEncoder(w.file)
	for _, agg := range aggregated {
		if err := encoder.Encode(agg); err != nil {
			return err
		}
	}
	return nil
}

func (w *JSONLWriter) Close() {
	if w.file != nil {
		w.file.Sync()
		w.file.Close()
	}
}
