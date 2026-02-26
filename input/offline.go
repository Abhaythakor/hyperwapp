package input

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Abhaythakor/hyperwapp/model"
	"github.com/Abhaythakor/hyperwapp/util"

	"github.com/Abhaythakor/hyperwapp/input/body"
	"github.com/Abhaythakor/hyperwapp/input/fff"
	"github.com/Abhaythakor/hyperwapp/input/katana"
	"github.com/Abhaythakor/hyperwapp/input/raw"
)

// OfflineFormat defines the type of offline input.
type OfflineFormat string

const (
	// FormatUnknown indicates an unidentified offline format.
	FormatUnknown OfflineFormat = "unknown"
	// FormatFFF indicates the fff tool output format.
	FormatFFF OfflineFormat = "fff"
	// FormatKatanaDir indicates a directory containing katana response files.
	FormatKatanaDir OfflineFormat = "katana-dir"
	// FormatKatanaFile indicates a single katana response file.
	FormatKatanaFile OfflineFormat = "katana-file"
	// FormatRawHTTP indicates a raw HTTP response dump.
	FormatRawHTTP OfflineFormat = "raw-http"
	// FormatBodyOnly indicates a file treated as a raw response body.
	FormatBodyOnly OfflineFormat = "body-only"
)

// DetectOfflineFormat identifies the format of the given path (file or directory).
func DetectOfflineFormat(path string) OfflineFormat {
	util.Debug("DetectOfflineFormat: Checking path: %s", path)
	fileInfo, err := os.Stat(path)
	if err != nil {
		util.Warn("DetectOfflineFormat: Could not stat path %s: %v", path, err)
		return FormatUnknown
	}

	if fileInfo.IsDir() {
		if IsFFFDirectory(path) {
			util.Debug("Detected FFF Directory: %s", path)
			return FormatFFF
		}
		if IsKatanaDirectory(path) {
			util.Debug("Detected Katana Directory: %s", path)
			return FormatKatanaDir
		}
		util.Debug("Directory %s is not FFF or Katana directory. Falling back to Body Only.", path)
		return FormatBodyOnly
	} else { // It's a file
		data, err := os.ReadFile(path) // Only call ReadFile if it's actually a file
		if err != nil {
			util.Warn("Failed to read file %s for format detection: %v", path, err)
			return FormatUnknown
		}

		if katana.IsKatanaFileContent(data) {
			util.Debug("Detected Katana File: %s", path)
			return FormatKatanaFile
		}
		if raw.IsRawHTTPContent(data) {
			util.Debug("Detected Raw HTTP File: %s", path)
			return FormatRawHTTP
		}
		util.Debug("Detected Body Only File: %s", path)
		return FormatBodyOnly
	}
}

// CountOffline performing a fast pass to count total targets without parsing.
// It uses concurrency to speed up discovery in large directory structures.
func CountOffline(path string, concurrency int) (uint32, error) {
	format := DetectOfflineFormat(path)
	var count atomic.Uint32

	fileInfo, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	if !fileInfo.IsDir() {
		return 1, nil
	}

	if format == FormatRawHTTP {
		return 1, nil
	}

	if concurrency <= 0 {
		concurrency = 1
	}

	var wg sync.WaitGroup

	// We walk the top level and spawn goroutines for each top-level entry
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}

	semaphore := make(chan struct{}, concurrency)

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			wg.Add(1)
			go func(p string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				_ = filepath.WalkDir(p, func(p2 string, d os.DirEntry, err error) error {
					if err == nil && !d.IsDir() && isTargetFile(d.Name(), format) {
						count.Add(1)
					}
					return nil
				})
			}(fullPath)
		} else {
			if isTargetFile(entry.Name(), format) {
				count.Add(1)
			}
		}
	}

	wg.Wait()
	return count.Load(), nil
}

func isTargetFile(fileName string, format OfflineFormat) bool {
	switch format {
	case FormatFFF:
		return strings.HasSuffix(fileName, ".headers")
	case FormatKatanaDir, FormatKatanaFile:
		return strings.Contains(fileName, ".txt")
	default:
		return true
	}
}

// ParseOffline dispatches the parsing to the correct handler based on detected format.
func ParseOffline(path string, skipFunc func(string) bool, concurrency int) (<-chan model.OfflineInput, error) {
	outputCh := make(chan model.OfflineInput)

	format := DetectOfflineFormat(path)
	if format == FormatUnknown { // If DetectOfflineFormat returns FormatUnknown, it means stat failed or it's genuinely unknown
		return nil, fmt.Errorf("invalid offline input path or unknown format: %s", path)
	}
	util.Debug("Detected offline format: %s for %s", format, path)

	go func() {
		defer close(outputCh)

		var inputSourceCh <-chan model.OfflineInput
		var parseErr error

		switch format {
		case FormatFFF:
			inputSourceCh, parseErr = fff.ParseFFF(path, skipFunc)
		case FormatKatanaDir:
			inputSourceCh, parseErr = katana.ParseKatanaDir(path, skipFunc, concurrency)
		case FormatKatanaFile:
			inputSourceCh, parseErr = katana.ParseKatanaFile(path, "", skipFunc)
		case FormatRawHTTP:
			inputSourceCh, parseErr = raw.ParseRawHTTP(path, skipFunc, concurrency)
		case FormatBodyOnly:
			inputSourceCh, parseErr = body.ParseBodyOnly(path, skipFunc, concurrency)
		case FormatUnknown:
			util.Warn("Unsupported or unknown offline format for path: %s", path)
			return // Exit goroutine
		default:
			util.Warn("Unhandled offline format: %s for path: %s", format, path)
			return // Exit goroutine
		}

		if parseErr != nil {
			util.Warn("Error during offline parsing of %s (format: %s): %v", path, format, parseErr)
			return // Exit goroutine
		}

		// Stream inputs from the source channel to the output channel
		for input := range inputSourceCh {
			outputCh <- input
		}
	}()

	return outputCh, nil
}

// IsFFFDirectory checks if a directory matches the fff output structure.
func IsFFFDirectory(path string) bool {
	util.Debug("IsFFFDirectory: Checking directory %s", path)
	foundHeaders := false
	foundBody := false

	// Check a few levels deep for FFF files
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Limit depth to avoid walking huge trees just for detection
			rel, _ := filepath.Rel(path, p)
			if rel != "." && strings.Count(filepath.ToSlash(rel), "/") > 2 {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".headers") {
			foundHeaders = true
		}
		if strings.HasSuffix(p, ".body") {
			foundBody = true
		}
		if foundHeaders && foundBody {
			return filepath.SkipAll // Found both, we are sure
		}
		return nil
	})
	return foundHeaders && foundBody
}

// IsKatanaDirectory checks if a directory matches the katana output structure.
func IsKatanaDirectory(path string) bool {
	util.Debug("IsKatanaDirectory: Checking directory %s", path)
	if _, err := os.Stat(filepath.Join(path, "index.txt")); err == nil {
		return true
	}

	foundTxt := false
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(path, p)
			if rel != "." && strings.Count(filepath.ToSlash(rel), "/") > 2 {
				return filepath.SkipDir
			}
			return nil
		}
		// Match Katana-like .txt files
		if strings.HasSuffix(p, ".txt") {
			// Check if it's a Katana file by content (first few bytes)
			data, err := os.Open(p)
			if err == nil {
				defer data.Close()
				head := make([]byte, 1024)
				n, _ := data.Read(head)
				if katana.IsKatanaFileContent(head[:n]) {
					foundTxt = true
					return filepath.SkipAll
				}
			}
		}
		return nil
	})
	return foundTxt
}
