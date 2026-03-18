package output

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"

	"github.com/Abhaythakor/hyperwapp/aggregate"
	"github.com/Abhaythakor/hyperwapp/model"
)

// JSONLWriter implements the Writer interface for JSON Lines output.
type JSONLWriter struct {
	file    *os.File
	buf     *bufio.Writer // Buffered IO
	encoder *json.Encoder
	mu      sync.Mutex
	mode    string
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

	bufferedWriter := bufio.NewWriterSize(file, 4*1024*1024) // 4MB RAM Buffer
	return &JSONLWriter{
		file:    file,
		buf:     bufferedWriter,
		encoder: json.NewEncoder(bufferedWriter),
		mode:    "all",
	}, nil
}

func (w *JSONLWriter) Write(detections []model.Detection) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, d := range detections {
		if err := w.encoder.Encode(d); err != nil {
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
	for _, agg := range aggregated {
		if err := w.encoder.Encode(agg); err != nil {
			return err
		}
	}
	return nil
}

func (w *JSONLWriter) Close() {
	if w.file != nil {
		w.mu.Lock()
		_ = w.buf.Flush() // Flush the 4MB RAM buffer
		_ = w.file.Sync()  // Force OS to write to physical disk
		_ = w.file.Close()
		w.mu.Unlock()
	}
}
