package online

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"hyperwapp/model"
	"hyperwapp/util"
)

// FetchOnline fetches the content of a URL and returns headers and body.
func FetchOnline(target model.Target, timeout int) (map[string][]string, []byte, error) {
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	req, err := http.NewRequest("GET", target.URL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request for %s: %w", target.URL, err)
	}
	req.Header.Set("User-Agent", "hyperwapp/1.0.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch %s: %w", target.URL, err)
	}
	defer resp.Body.Close()

	headers := make(map[string][]string)
	for k, v := range resp.Header {
		headers[k] = v
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		util.Warn("Failed to read body for %s: %v", target.URL, err)
		// Don't return error, proceed with headers if body read fails
		return headers, nil, nil
	}

	return headers, body, nil
}
