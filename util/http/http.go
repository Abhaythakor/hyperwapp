package http

// ExtractHost extracts the host from HTTP headers, falling back to a provided domain.
// It checks for "Host" and "host" headers.
func ExtractHost(headers map[string][]string, fallbackDomain string) string {
	if host, ok := headers["Host"]; ok && len(host) > 0 {
		return host[0]
	}
	if host, ok := headers["host"]; ok && len(host) > 0 { // Case-insensitivity
		return host[0]
	}
	return fallbackDomain
}
