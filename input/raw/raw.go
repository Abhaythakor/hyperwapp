package raw

import (
	"bufio"
	"bytes"
	"io" // Added import
	"os"
	"strings"

	"github.com/Abhaythakor/hyperwapp/model"
	"github.com/Abhaythakor/hyperwapp/util"
	"github.com/Abhaythakor/hyperwapp/util/http"
)

func ContainsHeaderLine(data []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Skip the first line, which might be an HTTP status line
	if scanner.Scan() {
		// Look for a line that looks like a header (e.g., "Key: Value")
		for scanner.Scan() {
			line := scanner.Bytes()
			if bytes.Contains(line, []byte(":")) {
				return true
			}
			if len(line) == 0 { // Stop at first blank line after status
				return false
			}
		}
	}
	return false
}

func IsRawHTTPContent(data []byte) bool {
	// A raw HTTP file must contain at least one HTTP status line and some headers.
	// It should NOT contain a request line (e.g., GET /) as that would indicate Katana.
	dataStr := string(data)
	if !strings.Contains(dataStr, "HTTP/1.") {
		return false
	}
	if strings.Contains(dataStr, "GET ") || strings.Contains(dataStr, "POST ") { // Likely a Katana file if contains request method
		return false
	}
	return ContainsHeaderLine(data)
}

// ParseRawHTTP parses a file containing one or more raw HTTP responses.
func ParseRawHTTP(path string) (<-chan model.OfflineInput, error) {
	outputCh := make(chan model.OfflineInput)

	go func() {
		defer close(outputCh)

		file, err := os.Open(path)
		if err != nil {
			util.Warn("Failed to open raw http file %s: %v", path, err)
			return // Exit goroutine
		}
		defer file.Close()

		rawResponseCh := splitHTTPResponses(file) // Now returns a channel of rawHTTPResponse

		for rawResp := range rawResponseCh {
			headers := parseRawHeaders(bytes.NewReader(rawResp.Headers)) // Pass io.Reader
			body := rawResp.Body
			domain := http.ExtractHost(headers, "unknown") // Try to infer domain from Host header

			outputCh <- model.OfflineInput{
				Domain:  domain,
				URL:     "", // URL cannot be reliably determined from raw response dump
				Headers: headers,
				Body:    body,
			}
			util.Debug("Created Raw HTTP OfflineInput for Domain: %s (URL unknown)", domain)
		}
	}()

	return outputCh, nil
}

// rawHTTPResponse represents a single raw HTTP response split into headers and body.
type rawHTTPResponse struct {
	Headers []byte
	Body    []byte
}

// splitHTTPResponses splits a byte slice containing one or more raw HTTP responses.
func splitHTTPResponses(r io.Reader) <-chan rawHTTPResponse {
	outputCh := make(chan rawHTTPResponse)

	go func() {
		defer close(outputCh)

		reader := bufio.NewReaderSize(r, 1024*1024) // 1MB buffer

		var currentResponse bytes.Buffer
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil && err != io.EOF {
				util.Warn("Error reading line from raw HTTP input: %v", err)
				break
			}

			// Check for the start of a new HTTP response (e.g., "HTTP/1.1 200 OK")
			// and if currentResponse already has content (i.e., not the very first line of the file)
			if bytes.HasPrefix(line, []byte("HTTP/1.")) && currentResponse.Len() > 0 {
				// Process the previous response before starting a new one
				resp := parseSingleHTTPResponse(bytes.NewReader(currentResponse.Bytes())) // Pass io.Reader
				if resp != nil {
					outputCh <- *resp
				}
				currentResponse.Reset() // Reset for the new response
			}

			currentResponse.Write(line)
			if err == io.EOF {
				break
			}
		}

		// Process the last response
		if currentResponse.Len() > 0 {
			resp := parseSingleHTTPResponse(bytes.NewReader(currentResponse.Bytes())) // Pass io.Reader
			if resp != nil {
				outputCh <- *resp
			}
		}
	}()
	return outputCh
}

// parseSingleHTTPResponse parses a single HTTP response block.
func parseSingleHTTPResponse(r io.Reader) *rawHTTPResponse {
	reader := bufio.NewReader(r)

	var headersBuffer bytes.Buffer
	var bodyBuffer bytes.Buffer
	inHeaders := true
	foundStatusLine := false

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			util.Warn("Error reading line from single HTTP response: %v", err)
			break
		}

		if !foundStatusLine && bytes.HasPrefix(line, []byte("HTTP/1.")) {
			foundStatusLine = true
		}

		if inHeaders {
			if len(line) == 1 && line[0] == '\n' || len(line) == 2 && line[0] == '\r' && line[1] == '\n' { // Blank line, end of headers
				inHeaders = false
			}
			headersBuffer.Write(line)
		} else {
			bodyBuffer.Write(line)
		}

		if err == io.EOF {
			break
		}
	}

	if !foundStatusLine || headersBuffer.Len() == 0 { // Must have at least a status line and some headers
		return nil
	}

	return &rawHTTPResponse{
		Headers: headersBuffer.Bytes(),
		Body:    bodyBuffer.Bytes(),
	}
}

// parseRawHeaders parses raw HTTP headers into a map.
func parseRawHeaders(r io.Reader) map[string][]string {
	headers := make(map[string][]string)
	reader := bufio.NewReader(r) // Use bufio.Reader directly
	isFirstLine := true
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			util.Warn("Error reading raw header line: %v", err)
			break
		}
		trimmedLine := strings.TrimSpace(line)

		if isFirstLine {
			// Skip status line like "HTTP/1.1 200 OK"
			if strings.HasPrefix(trimmedLine, "HTTP/") {
				isFirstLine = false
				if err == io.EOF {
					break
				}
				continue
			}
			isFirstLine = false
		}

		if len(trimmedLine) == 0 { // End of headers
			break
		}

		parts := strings.SplitN(trimmedLine, ":", 2)
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
