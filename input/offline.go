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
	"github.com/Abhaythakor/hyperwapp/input/custom"
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
	// FormatCustom indicates an input parsed with a YAML config (json or regex).
	FormatCustom OfflineFormat = "custom"
)

// DetectOfflineFormat identifies the format of the given path (file or directory).
func DetectOfflineFormat(path string, hasCustomConfig bool) OfflineFormat {
	if hasCustomConfig {
		return FormatCustom
	}
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

		// Check if it's a directory of JSON files
		isJSONDir := false
		_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && (strings.HasSuffix(strings.ToLower(d.Name()), ".json") || strings.HasSuffix(strings.ToLower(d.Name()), ".jsonl")) {
				isJSONDir = true
				return filepath.SkipAll
			}
			return nil
		})
		if isJSONDir {
			util.Debug("Detected JSON Directory: %s", path)
			return FormatCustom
		}

		// Fallback to Body Only
		util.Debug("Directory %s is not FFF or Katana directory. Falling back to Body Only.", path)
		return FormatBodyOnly
	} else { // It's a file
		f, err := os.Open(path)
		if err != nil {
			util.Warn("Failed to open file %s for format detection: %v", path, err)
			return FormatUnknown
		}
		defer f.Close()

		// Only read the first 4KB for detection
		data := make([]byte, 4096)
		n, _ := f.Read(data)
		data = data[:n]

		if katana.IsKatanaFileContent(data) {
			util.Debug("Detected Katana File: %s", path)
			return FormatKatanaFile
		}
		if raw.IsRawHTTPContent(data) {
			util.Debug("Detected Raw HTTP File: %s", path)
			return FormatRawHTTP
		}

		// JSON/JSONL auto-detection
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".json" || ext == ".jsonl" || (len(data) > 0 && (data[0] == '{' || data[0] == '[')) {
			util.Debug("Detected JSON File: %s", path)
			return FormatCustom
		}

		util.Debug("Detected Body Only File: %s", path)
		return FormatBodyOnly
	}
}

// CountOffline performing a fast pass to count total targets without parsing.
func CountOffline(path string, concurrency int, hasCustomConfig bool) (uint32, error) {
	format := DetectOfflineFormat(path, hasCustomConfig)
	var count atomic.Uint32

	fileInfo, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	if !fileInfo.IsDir() {
		if format == FormatCustom {
			// For custom formats, we must count records/lines.
			// This is tricky if it's regex blocks, but for now we'll assume line-based or a simple estimate.
			return countLines(path)
		}
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
						if format == FormatCustom {
							lc, _ := countLines(p2)
							count.Add(lc)
						} else {
							count.Add(1)
						}
					}
					return nil
				})
			}(fullPath)
		} else {
			if isTargetFile(entry.Name(), format) {
				if format == FormatCustom {
					lc, _ := countLines(fullPath)
					count.Add(lc)
				} else {
					count.Add(1)
				}
			}
		}
	}

	wg.Wait()
	return count.Load(), nil
}

func countLines(path string) (uint32, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var count uint32
	// Use a 1MB buffer for reading
	buf := make([]byte, 1024*1024)
	var lastChar byte
	for {
		n, err := file.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				if buf[i] == '\n' {
					count++
				}
			}
			lastChar = buf[n-1]
		}
		if err != nil {
			break
		}
	}
	
	// If the file has content but doesn't end with a newline, count the last line
	if count == 0 {
		fi, _ := os.Stat(path)
		if fi.Size() > 0 {
			return 1, nil
		}
	} else if lastChar != '\n' {
		count++
	}

	return count, nil
}

func isTargetFile(fileName string, format OfflineFormat) bool {
	switch format {
	case FormatFFF:
		return strings.HasSuffix(fileName, ".headers")
	case FormatKatanaDir, FormatKatanaFile:
		return strings.Contains(fileName, ".txt")
	case FormatCustom:
		return true // Configured to handle any file
	default:
		return true
	}
}

// ParseOffline dispatches the parsing to the correct handler based on detected format.
func ParseOffline(path string, skipFunc func(string) bool, concurrency int, customCfg *custom.CompiledConfig) (<-chan model.OfflineInput, error) {
	outputCh := make(chan model.OfflineInput, 1000) // Buffer the bridge

	format := DetectOfflineFormat(path, customCfg != nil)
	if format == FormatUnknown {
		return nil, fmt.Errorf("invalid offline input path or unknown format: %s", path)
	}
	util.Debug("Detected offline format: %s for %s", format, path)

	go func() {
		defer close(outputCh)

		var inputSourceCh <-chan model.OfflineInput
		var parseErr error

		switch format {
		case FormatCustom:
			inputSourceCh, parseErr = custom.ParseCustom(path, customCfg, skipFunc, concurrency)
		case FormatFFF:
			inputSourceCh, parseErr = fff.ParseFFF(path, skipFunc)
		case FormatKatanaDir:
			inputSourceCh, parseErr = katana.ParseKatanaDir(path, skipFunc, concurrency)
		case FormatKatanaFile:
			var inputs []model.OfflineInput
			inputs, parseErr = katana.ParseKatanaFile(path, "", skipFunc)
			if parseErr == nil {
				ch := make(chan model.OfflineInput, len(inputs))
				for _, in := range inputs {
					ch <- in
				}
				close(ch)
				inputSourceCh = ch
			}
		case FormatRawHTTP:
			inputSourceCh, parseErr = raw.ParseRawHTTP(path, skipFunc, concurrency)
		case FormatBodyOnly:
			inputSourceCh, parseErr = body.ParseBodyOnly(path, skipFunc, concurrency)
		default:
			util.Warn("Unsupported offline format: %s", format)
			return
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
