package input

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Abhaythakor/hyperwapp/model"
)

// normalizeTarget validates and normalizes a given input string into a Target struct.
func normalizeTarget(input string, offlineMode bool) (model.Target, error) {
	if offlineMode {
		// For offline mode, the input is treated as a file path or directory.
		// Domain and URL will be determined during offline parsing.
		return model.Target{URL: input}, nil // Store the path in the URL field temporarily
	}

	// Online mode: Expect a valid URL
	u, err := url.Parse(input)
	if err != nil {
		return model.Target{}, fmt.Errorf("invalid URL: %w", err)
	}

	if !u.IsAbs() || (u.Scheme != "http" && u.Scheme != "https") {
		return model.Target{}, fmt.Errorf("URL must be absolute and start with http:// or https://")
	}

	domain := u.Hostname()
	if strings.HasPrefix(domain, "www.") {
		domain = domain[4:]
	}

	return model.Target{URL: u.String(), Domain: domain}, nil
}
