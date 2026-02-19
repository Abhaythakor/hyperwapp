package util

import (
	"bufio"
	"fmt"
	"os"
	"sync"
)

// ResumeManager handles tracking completed targets for resuming scans.
type ResumeManager struct {
	mu         sync.RWMutex
	file       *os.File
	completed  map[string]struct{}
	filePath   string
	enabled    bool
	TotalCount uint32 // Loaded total from previous run
}

// NewResumeManager initializes the manager and loads existing progress if resume is enabled.
func NewResumeManager(path string, enabled bool) (*ResumeManager, error) {
	rm := &ResumeManager{
		completed: make(map[string]struct{}),
		filePath:  path,
		enabled:   enabled,
	}

	if !enabled {
		return rm, nil
	}

	// Load existing progress
	if _, err := os.Stat(path); err == nil {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(file)
		firstLine := true
		for scanner.Scan() {
			text := scanner.Text()
			if firstLine {
				firstLine = false
				// Try to parse total count from first line
				var total uint32
				_, err := fmt.Sscanf(text, "TOTAL:%d", &total)
				if err == nil {
					rm.TotalCount = total
					continue
				}
			}
			rm.completed[text] = struct{}{}
		}
		file.Close()
	}

	// Open for appending
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	rm.file = file

	return rm, nil
}

// SaveTotal saves the total count to the resume file (only if file is empty/new).
func (rm *ResumeManager) SaveTotal(total uint32) {
	if !rm.enabled || rm.file == nil {
		return
	}
	// We only write total if we didn't load one
	if rm.TotalCount == 0 {
		rm.mu.Lock()
		defer rm.mu.Unlock()
		_, _ = rm.file.WriteString(fmt.Sprintf("TOTAL:%d\n", total))
	}
}

// IsCompleted checks if a target ID has already been processed.
func (rm *ResumeManager) IsCompleted(id string) bool {
	if !rm.enabled {
		return false
	}
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	_, ok := rm.completed[id]
	return ok
}

// MarkCompleted saves a target ID to the resume file.
func (rm *ResumeManager) MarkCompleted(id string) {
	if !rm.enabled || rm.file == nil {
		return
	}
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.completed[id] = struct{}{}
	_, _ = rm.file.WriteString(id + "\n")
}

// Close closes the resume file.
func (rm *ResumeManager) Close() {
	if rm.file != nil {
		_ = rm.file.Sync()
		_ = rm.file.Close()
	}
}

// Cleanup deletes the resume file (usually called when a scan finishes successfully).
func (rm *ResumeManager) Cleanup() {
	rm.Close()
	if rm.enabled {
		_ = os.Remove(rm.filePath)
	}
}
