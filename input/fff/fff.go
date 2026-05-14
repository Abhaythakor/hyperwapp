package fff

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync" // Added import

	"github.com/Abhaythakor/hyperwapp/model"
	"github.com/Abhaythakor/hyperwapp/util"
)

// ParseFFF parses an fff directory structure and returns a channel of OfflineInput.
func ParseFFF(ctx context.Context, root string, skipFunc func(string) bool, concurrency int) (<-chan *model.OfflineInput, error) {
	outputCh := make(chan *model.OfflineInput, 1000)

	if concurrency <= 0 {
		concurrency = 1
	}

	go func() {
		defer close(outputCh)

		util.Debug("Walking FFF directory: %s", root)
		
		// List top-level domain directories
		entries, err := os.ReadDir(root)
		if err != nil {
			util.Warn("Failed to read FFF root: %v", err)
			return
		}

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, concurrency)

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			
			domain := entry.Name()
			domainPath := filepath.Join(root, domain)

			wg.Add(1)
			select {
			case <-ctx.Done():
				wg.Done()
				return
			case semaphore <- struct{}{}:
				go func(dPath, dom string) {
					defer wg.Done()
					defer func() { <-semaphore }()
					
					// Map to group hash -> {headers, body} within THIS domain only
					filesByHash := make(map[string]map[string]string)
					
					_ = filepath.WalkDir(dPath, func(path string, d os.DirEntry, err error) error {
						if err != nil || d.IsDir() {
							return nil
						}

						if strings.HasSuffix(path, ".headers") || strings.HasSuffix(path, ".body") {
							hash := extractHash(filepath.Base(path))
							if hash == "" {
								return nil
							}

							if _, ok := filesByHash[hash]; !ok {
								filesByHash[hash] = make(map[string]string)
							}

							if strings.HasSuffix(path, ".headers") {
								filesByHash[hash]["headers"] = path
							} else {
								filesByHash[hash]["body"] = path
							}
						}
						return nil
					})

					// Once this domain is walked, stream its results immediately
					for _, files := range filesByHash {
						select {
						case <-ctx.Done():
							return
						default:
							// RESUME CHECK
							if skipFunc != nil {
								hPath, hasHeaders := files["headers"]
								if hasHeaders {
									if skipFunc(hPath) {
										input := model.OfflineInputPool.Get().(*model.OfflineInput)
										input.Reset()
										input.Path = hPath
										input.Skipped = true
										outputCh <- input
										continue
									}
								} else if bPath, ok := files["body"]; ok {
									if skipFunc(bPath) {
										input := model.OfflineInputPool.Get().(*model.OfflineInput)
										input.Reset()
										input.Path = bPath
										input.Skipped = true
										outputCh <- input
										continue
									}
								}
							}

							inputs := buildFFFInputsFromGroup(files, dPath, dom)
							for _, input := range inputs {
								outputCh <- input
							}
						}
					}
				}(domainPath, domain)
			}
		}
		wg.Wait()
	}()

	return outputCh, nil
}

// buildFFFInputsFromGroup constructs OfflineInput objects from a single grouped fff files map.
func buildFFFInputsFromGroup(files map[string]string, root, domain string) []*model.OfflineInput {
	input := model.OfflineInputPool.Get().(*model.OfflineInput)
	input.Reset()

	var fileURLPath string // To derive the URL correctly
	var sourcePath string

	if hPath, ok := files["headers"]; ok {
		sourcePath = hPath
		err := parseHeadersFile(hPath, input.Headers)
		if err != nil {
			util.Warn("Error parsing fff headers file %s: %v", hPath, err)
		}
		fileURLPath = hPath
	}
	if bPath, ok := files["body"]; ok {
		if sourcePath == "" {
			sourcePath = bPath
		}
		b, err := os.ReadFile(bPath)
		if err != nil {
			util.Warn("Error reading fff body file %s: %v", bPath, err)
		} else {
			input.Body = b
		}
		if fileURLPath == "" { // If only body file exists, use its path for URL
			fileURLPath = bPath
		}
	}

	url := DeriveURL(root, fileURLPath, domain)
	input.Domain = domain
	input.URL = url
	input.Path = sourcePath

	util.Debug("Created FFF OfflineInput for URL: %s (Domain: %s)", url, domain)
	return []*model.OfflineInput{input}
}

// parseHeadersFile parses an fff .headers file into the provided headers map.
func parseHeadersFile(path string, headers map[string][]string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open headers file %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Read the first line and check for HTTP status
	if scanner.Scan() {
		firstLine := scanner.Text()
		if !strings.HasPrefix(firstLine, "HTTP/") {
			// If it's not an HTTP status line, it must be the first header. Process it.
			parts := strings.SplitN(firstLine, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				headers[key] = append(headers[key], value)
			}
		}
		// If it is an HTTP status line, we just discard it.
		// The loop below will then process the next lines.
	}

	// Process all remaining lines as headers
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headers[key] = append(headers[key], value)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading headers file %s: %w", path, err)
	}

	return nil
}

// DeriveURL constructs a URL from the fff root, file path, and domain.
func DeriveURL(domainRoot, filePath, domain string) string {
	if filePath == "" {
		return "" // No file path to derive from
	}
	rel, err := filepath.Rel(domainRoot, filepath.Dir(filePath))
	if err != nil {
		util.Warn("Error deriving relative path for %s from %s: %v", filePath, domainRoot, err)
		return "https://" + domain // Fallback
	}

	if rel == "." { // File is directly in the domain root
		return "https://" + domain
	}
	return "https://" + domain + "/" + filepath.ToSlash(rel)
}

// extractHash extracts the hash prefix from an fff filename.
// e.g., "cb22c4cf4192095fa403af8695acf42f28ffe7ad.body" -> "cb22c4cf4192095fa403af8695acf42f28ffe7ad"
func extractHash(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return ""
	}
	base := filename[:idx]
	// Check if it looks like a hash (e.g., length 32-64 for SHA1/SHA256)
	if len(base) >= 32 && len(base) <= 64 && isHex(base) { // Common hash lengths
		return base
	}
	return ""
}

func isHex(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}
