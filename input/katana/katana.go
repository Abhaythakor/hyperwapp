package katana

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url" // Added import
	"os"
	"path/filepath"
	"strings"

	"hyperwapp/model"
	"hyperwapp/util" // Added import for util package
	"hyperwapp/util/http"
)

func IsKatanaFileContent(data []byte) bool {
	// Katana files usually contain both a request line (e.g., GET / HTTP/1.1)
	// and a response status line (e.g., HTTP/1.1 200 OK).
	// We'll check for these minimal indicators without doing a full parse.
	dataStr := string(data)
	return strings.Contains(dataStr, "GET ") && strings.Contains(dataStr, "HTTP/1.")
}

// ParseKatanaDir parses a katana output directory and returns a channel of OfflineInput.
func ParseKatanaDir(root string) (<-chan model.OfflineInput, error) {
	outputCh := make(chan model.OfflineInput)

	go func() {
		defer close(outputCh)

		util.Debug("Walking Katana directory recursively: %s", root)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				util.Warn("Error walking Katana directory at %s: %v", path, err)
				return nil
			}

			if d.IsDir() {
				return nil
			}

			fileName := d.Name()
			// Match .txt or files containing .txt (like .txt.~1~)
			if strings.Contains(fileName, ".txt") {
				// Infer domain from the immediate parent directory name
				parentDir := filepath.Base(filepath.Dir(path))
				domain := parentDir
				if domain == "." || domain == "responses" || domain == "katana-output" {
					domain = "" // Let the parser try to extract it from headers
				}

				inputCh, err := ParseKatanaFile(path, domain)
				if err != nil {
					util.Warn("Error parsing katana file %s: %v", path, err)
					return nil
				}
				for input := range inputCh {
					outputCh <- input
				}
			}
			return nil
		})

		if err != nil {
			util.Warn("Failed to complete recursive walk of Katana directory %s: %v", root, err)
		}
	}()

	return outputCh, nil
}

// ParseKatanaFile parses a single katana response file and returns a channel of OfflineInput.
func ParseKatanaFile(path, fallbackDomain string) (<-chan model.OfflineInput, error) {
	outputCh := make(chan model.OfflineInput, 1) // Buffered channel for a single input

	go func() {
		defer close(outputCh)

		file, err := os.Open(path)
		if err != nil {
			util.Warn("Failed to open katana file %s: %v", path, err)
			return // Exit goroutine
		}
		defer file.Close()

		parts := splitKatanaRequestResponse(file)
		if parts == nil {
			util.Warn("Malformed katana file (could not split request/response): %s", path)
			return // Exit goroutine
		}

		requestHeaders := parseRequestHeadersKatana(bytes.NewReader(parts.RequestHeaders))
		responseHeaders := parseResponseHeadersKatana(bytes.NewReader(parts.ResponseHeaders))
		body := parts.Body
		domain := http.ExtractHost(requestHeaders, fallbackDomain)
		url := reconstructURLKatana(parts.RequestLine, requestHeaders, domain, parts.InitialURL)

		util.Debug("Created Katana OfflineInput for URL: %s (Domain: %s)", url, domain)
		outputCh <- model.OfflineInput{
			Domain:  domain,
			URL:     url,
			Headers: responseHeaders,
			Body:    body,
		}
	}()

	return outputCh, nil
}

// katanaParts represents the split sections of a katana response file.
type katanaParts struct {
	InitialURL      []byte
	RequestLine     []byte
	RequestHeaders  []byte
	ResponseStatus  []byte
	ResponseHeaders []byte
	Body            []byte
}

// readUntilDelimiter reads bytes from r until the delimiter is found or EOF.
// It returns the bytes read before the delimiter and any error.
// The delimiter is consumed from the reader.
func readUntilDelimiter(r *bufio.Reader, delimiter []byte) ([]byte, error) {
	var buffer bytes.Buffer
	for {
		line, err := r.ReadBytes('\n')
		buffer.Write(line)

		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if bytes.HasSuffix(buffer.Bytes(), delimiter) {
			break
		}
	}
	return buffer.Bytes(), nil
}

// splitKatanaRequestResponse splits a katana file into its constituent parts using an io.Reader.
func splitKatanaRequestResponse(r io.Reader) *katanaParts {
	reader := bufio.NewReaderSize(r, 1024*1024) // 1MB buffer

	parts := &katanaParts{}
	var err error // Declare err once at the top

	// Explicitly declare these local variables for clarity and to avoid scope issues.
	var rawRequestBlock []byte
	var rawResponseBlock []byte

	// Step 1: Try to read the first line as a potential initial URL.
	// Katana files often have the URL on the first line, followed by a blank line, then the request.
	firstLine, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		util.Warn("Error reading first line of katana file: %v", err)
		return nil
	}
	trimmedFirstLine := strings.TrimSpace(firstLine)

	isPotentialInitialURL := (strings.HasPrefix(trimmedFirstLine, "http://") || strings.HasPrefix(trimmedFirstLine, "https://"))

	if isPotentialInitialURL {
		// If it looks like a URL, try to read the next line. It should be blank.
		secondLine, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			util.Warn("Error reading second line after potential initial URL in katana file: %v", err)
			// Not fatal, proceed as if the first line was not an initial URL
		}

		if strings.TrimSpace(secondLine) == "" { // Confirmed initial URL + blank line pattern
			parts.InitialURL = []byte(trimmedFirstLine)
		} else {
			// If second line is not blank, the first line was probably part of the request.
			// We need to "un-read" both lines. This is tricky with bufio.Reader.
			// A simpler way: we've already consumed these, so reconstruct a reader with these lines
			// prepended, and pass to the next stage.
			// For now, if the second line is not blank, we treat the first line (trimmedFirstLine)
			// as part of the rawRequestBlock, and will feed it into the next part of parsing.
			// So, we'll create a new reader that starts with these two lines + the original reader's content.
			var tempBuffer bytes.Buffer
			tempBuffer.WriteString(firstLine)
			tempBuffer.WriteString(secondLine)
			newReader := bufio.NewReaderSize(io.MultiReader(&tempBuffer, reader), 1024*1024)
			reader = newReader // Continue reading from this new reader
		}
	} else {
		// If the first line is not a URL, it's likely the request line.
		// We need to push it back into the reader to be parsed by readUntilDelimiter.
		// For simplicity, we'll create a new reader with this line prepended.
		var tempBuffer bytes.Buffer
		tempBuffer.WriteString(firstLine)
		newReader := bufio.NewReaderSize(io.MultiReader(&tempBuffer, reader), 1024*1024)
		reader = newReader // Continue reading from this new reader
	}

	// Step 2: Read until the first double newline (request block separator)
	rawRequestBlock, err = readUntilDelimiter(reader, []byte("\n\n"))
	if err != nil {
		util.Warn("Error reading request block from katana file: %v", err)
		return nil
	}
	// Adjust block length based on delimiter
	rawRequestBlock = trimDelimiter(rawRequestBlock)

	requestBlockReader := bufio.NewReader(bytes.NewReader(rawRequestBlock))
	firstLineReq, err := requestBlockReader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		util.Warn("Error reading request line from katana file: %v", err)
		return nil
	}
	parts.RequestLine = bytes.TrimSpace(firstLineReq)
	parts.RequestHeaders, _ = io.ReadAll(requestBlockReader) // Read remaining as headers

	// Step 3: Read until the second double newline (response block separator)
	rawResponseBlock, err = readUntilDelimiter(reader, []byte("\n\n"))
	if err != nil {
		util.Warn("Error reading response block from katana file: %v", err)
		return nil
	}
	// Adjust block length based on delimiter
	rawResponseBlock = trimDelimiter(rawResponseBlock)

	responseBlockReader := bufio.NewReader(bytes.NewReader(rawResponseBlock))
	statusLine, err := responseBlockReader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		util.Warn("Error reading response status line from katana file: %v", err)
		return nil
	}
	parts.ResponseStatus = bytes.TrimSpace(statusLine)
	parts.ResponseHeaders, _ = io.ReadAll(responseBlockReader) // Read remaining as headers

	// Step 4: The remaining content is the body
	parts.Body, err = io.ReadAll(reader)
	if err != nil && err != io.EOF {
		util.Warn("Error reading body from katana file: %v", err)
	}

	return parts
}

// trimDelimiter removes the trailing \n\n or \r\n\r\n from a byte slice.
func trimDelimiter(data []byte) []byte {
	if bytes.HasSuffix(data, []byte("\r\n\r\n")) {
		return data[:len(data)-4]
	} else if bytes.HasSuffix(data, []byte("\n\n")) {
		return data[:len(data)-2]
	}
	return data
}


// parseResponseHeadersKatana parses raw response headers into a map.
func parseResponseHeadersKatana(r io.Reader) map[string][]string {
	headers := make(map[string][]string)
	reader := bufio.NewReader(r) // Use bufio.Reader directly

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			util.Warn("Error reading response header line: %v", err)
			break
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 { // End of headers
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headers[key] = append(headers[key], value)
		}
		if err == io.EOF {
			break
		}
	}
	return headers
}



// parseRequestHeadersKatana parses raw request headers into a map.
func parseRequestHeadersKatana(r io.Reader) map[string][]string {
	headers := make(map[string][]string)
	reader := bufio.NewReader(r) // Use bufio.Reader directly

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			util.Warn("Error reading request header line: %v", err)
			break
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 { // End of headers
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headers[key] = append(headers[key], value)
		}
		if err == io.EOF {
			break
		}
	}
	return headers
}

// reconstructURLKatana reconstructs the URL from the request line, request headers, and an optional initial URL.
func reconstructURLKatana(requestLine []byte, requestHeaders map[string][]string, domainHint string, initialURL []byte) string {
	if len(initialURL) > 0 {
		return string(initialURL) // Use the directly provided URL if available
	}

	line := string(requestLine)
	parts := strings.Split(line, " ")
	if len(parts) < 2 {
		return ""
	}
	rawPath := parts[1] // e.g., /path/to/resource or https://example.com/path

	scheme := "https" // Default to https
	host := ""
	path := ""

	// Check if the rawPath is an absolute URL itself
	if strings.HasPrefix(rawPath, "http://") || strings.HasPrefix(rawPath, "https://") {
		u, err := url.Parse(rawPath)
		if err == nil {
			scheme = u.Scheme
			host = u.Host
			path = u.Path
			if u.RawQuery != "" {
				path = path + "?" + u.RawQuery
			}
			return u.String() // If it's a full URL, return it directly.
		}
	}

	// Otherwise, try to get host from headers
	if h, ok := requestHeaders["Host"]; ok && len(h) > 0 {
		host = h[0]
	} else if h, ok := requestHeaders["host"]; ok && len(h) > 0 { // Case-insensitivity
		host = h[0]
	} else {
		host = domainHint // Fallback to domain hint from directory structure
	}

	// Determine path if not already an absolute URL
	path = rawPath
	if !strings.HasPrefix(path, "/") {
		// If path doesn't start with '/', it might be a relative path or missing
		// For robustness, ensure it starts with a slash or is empty
		path = "/" + path
	}

	if host == "" {
		return "" // Cannot construct URL without a host
	}

	return fmt.Sprintf("%s://%s%s", scheme, host, path)
}