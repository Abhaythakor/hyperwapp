package fff

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync" // Added import

	"github.com/Abhaythakor/hyperwapp/model"
	"github.com/Abhaythakor/hyperwapp/util"
)

// ParseFFF parses an fff directory structure and returns a channel of OfflineInput.
func ParseFFF(root string) (<-chan model.OfflineInput, error) {
	outputCh := make(chan model.OfflineInput)

	go func() {
		defer close(outputCh)

		// Grouping: Domain -> Hash -> map[type]path
		filesByDomainAndHash := make(map[string]map[string]map[string]string)
		// We also need to keep track of the "domain root" (where the domain starts) for URL derivation
		domainRoots := make(map[string]string)

		util.Debug("Walking FFF directory recursively: %s", root)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				util.Warn("Error walking FFF directory at %s: %v", path, err)
				return nil
			}
			if d.IsDir() {
				return nil
			}

			if strings.HasSuffix(path, ".headers") || strings.HasSuffix(path, ".body") {
				hash := extractHash(filepath.Base(path))
				if hash == "" {
					return nil
				}

				// Determine domain: The first directory under 'root' is our best guess for domain
				rel, _ := filepath.Rel(root, path)
				parts := strings.Split(filepath.ToSlash(rel), "/")
				if len(parts) < 2 {
					return nil // Should be root/domain/file or deeper
				}
				domain := parts[0]
				domainRoot := filepath.Join(root, domain)

				if _, ok := filesByDomainAndHash[domain]; !ok {
					filesByDomainAndHash[domain] = make(map[string]map[string]string)
					domainRoots[domain] = domainRoot
				}
				if _, ok := filesByDomainAndHash[domain][hash]; !ok {
					filesByDomainAndHash[domain][hash] = make(map[string]string)
				}

				if strings.HasSuffix(path, ".headers") {
					filesByDomainAndHash[domain][hash]["headers"] = path
				} else {
					filesByDomainAndHash[domain][hash]["body"] = path
				}
			}
			return nil
		})

		if err != nil {
			util.Warn("Failed to walk FFF directory %s: %v", root, err)
			return
		}

		// Now process all grouped files
		var wg sync.WaitGroup
		for domain, hashes := range filesByDomainAndHash {
			dRoot := domainRoots[domain]
			for _, files := range hashes {
				wg.Add(1)
				go func(f map[string]string, dr string, dom string) {
					defer wg.Done()
					inputs := buildFFFInputsFromGroup(f, dr, dom)
					for _, input := range inputs {
						outputCh <- input
					}
				}(files, dRoot, domain)
			}
		}
		wg.Wait()
	}()

	return outputCh, nil
}

// parseFFFDomain walks a domain's directory within an fff structure and builds OfflineInput objects,
// sending them to a channel.
func parseFFFDomain(path, domain string) (<-chan model.OfflineInput, error) {
	outputCh := make(chan model.OfflineInput)

	// Use a wait group to ensure all goroutines finish before closing the channel
	var wg sync.WaitGroup

	// anonymous goroutine to handle walking the directory and sending inputs
	go func() {
		defer close(outputCh)

		filesByHash := make(map[string]map[string]string) // hash -> type (headers/body) -> filepath

		err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				util.Warn("Error walking fff domain directory %s: %v", p, err)
				return err // Return the error to stop walking this subdirectory
			}
			if d.IsDir() {
				return nil
			}

			if strings.HasSuffix(p, ".headers") || strings.HasSuffix(p, ".body") {
				hash := extractHash(filepath.Base(p))
				if hash == "" {
					return nil // Skip files without a recognizable hash
				}

				if _, ok := filesByHash[hash]; !ok {
					filesByHash[hash] = make(map[string]string)
				}
				if strings.HasSuffix(p, ".headers") {
					filesByHash[hash]["headers"] = p
				}
				if strings.HasSuffix(p, ".body") {
					filesByHash[hash]["body"] = p
				}
			}
			return nil
		})

		if err != nil {
			util.Warn("Failed to walk fff domain directory %s: %v", path, err)
			return // Exit goroutine
		}

		// Now process the grouped files and send to outputCh
		for _, files := range filesByHash {
			wg.Add(1)
			go func(files map[string]string) {
				defer wg.Done()
				inputs := buildFFFInputsFromGroup(files, path, domain)
				for _, input := range inputs {
					outputCh <- input
				}
			}(files)
		}
		wg.Wait()
	}()

	return outputCh, nil
}

// buildFFFInputsFromGroup constructs OfflineInput objects from a single grouped fff files map.
func buildFFFInputsFromGroup(files map[string]string, root, domain string) []model.OfflineInput {
	var inputs []model.OfflineInput

	headers := make(map[string][]string)
	var body []byte
	var fileURLPath string // To derive the URL correctly

	if hPath, ok := files["headers"]; ok {
		parsedHeaders, err := parseHeadersFile(hPath)
		if err != nil {
			util.Warn("Error parsing fff headers file %s: %v", hPath, err)
		} else {
			headers = parsedHeaders
		}
		fileURLPath = hPath
	}
	if bPath, ok := files["body"]; ok {
		b, err := os.ReadFile(bPath)
		if err != nil {
			util.Warn("Error reading fff body file %s: %v", bPath, err)
		} else {
			body = b
		}
		if fileURLPath == "" { // If only body file exists, use its path for URL
			fileURLPath = bPath
		}
	}

	url := DeriveURL(root, fileURLPath, domain)

	inputs = append(inputs, model.OfflineInput{
		Domain:  domain,
		URL:     url,
		Headers: headers,
		Body:    body,
	})
	util.Debug("Created FFF OfflineInput for URL: %s (Domain: %s)", url, domain)
	return inputs
}

// parseHeadersFile parses an fff .headers file into an http.Header map.
func parseHeadersFile(path string) (map[string][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open headers file %s: %w", path, err)
	}
	defer file.Close()

	headers := make(map[string][]string)
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
		return nil, fmt.Errorf("error reading headers file %s: %w", path, err)
	}

	return headers, nil
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
