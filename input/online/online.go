package online

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Abhaythakor/hyperwapp/model"
	"github.com/Abhaythakor/hyperwapp/util"
)

var (
	defaultClient *http.Client
	once          sync.Once
)

// GetClient returns a shared HTTP client configured for high-concurrency scanning.
func GetClient(timeout int) *http.Client {
	once.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			MaxIdleConns:        500,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableKeepAlives:   false,
		}
		defaultClient = &http.Client{
			Transport: transport,
			Timeout:   time.Duration(timeout) * time.Second,
		}
	})
	return defaultClient
}

// FetchOnline fetches the content of a URL and returns headers and body.
func FetchOnline(ctx context.Context, target model.Target, timeout int) (map[string][]string, []byte, error) {
	client := GetClient(timeout)

	req, err := http.NewRequestWithContext(ctx, "GET", target.URL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request for %s: %w", target.URL, err)
	}
	req.Header.Set("User-Agent", "github.com/Abhaythakor/hyperwapp/1.0.0")

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
