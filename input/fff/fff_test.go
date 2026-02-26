package fff_test

import (
	"strings"
	"testing"

	"github.com/Abhaythakor/hyperwapp/input/fff" // Import the fff package
	"github.com/Abhaythakor/hyperwapp/model"
)

func TestParseFFFBasic(t *testing.T) {
	testRoot := "../../testdata/fff/simple"

	inputsCh, err := fff.ParseFFF(testRoot, nil)
	if err != nil {
		t.Fatalf("ParseFFF failed: %v", err)
	}

	var inputs []model.OfflineInput
	for in := range inputsCh {
		inputs = append(inputs, in)
	}

	if len(inputs) != 1 {
		t.Fatalf("Expected 1 OfflineInput, got %d", len(inputs))
	}

	input := inputs[0]
	if input.Domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", input.Domain)
	}
	if input.URL != "https://example.com" { // As it's in the root of domain dir
		t.Errorf("Expected URL 'https://example.com', got '%s'", input.URL)
	}
	if len(input.Headers) == 0 {
		t.Errorf("Expected headers, got none")
	}
	if !strings.Contains(string(input.Body), "Hello World") {
		t.Errorf("Expected body to contain 'Hello World'")
	}

	if serverHeader, ok := input.Headers["Server"]; !ok || serverHeader[0] != "nginx/1.18.0" {
		t.Errorf("Expected Server header 'nginx/1.18.0', got '%v'", serverHeader)
	}
}

func TestParseFFFHeadersOnly(t *testing.T) {
	testRoot := "../../testdata/fff/headers-only"
	inputsCh, err := fff.ParseFFF(testRoot, nil)
	if err != nil {
		t.Fatalf("ParseFFF failed: %v", err)
	}

	var inputs []model.OfflineInput
	for in := range inputsCh {
		inputs = append(inputs, in)
	}

	if len(inputs) != 1 {
		t.Fatalf("Expected 1 OfflineInput, got %d", len(inputs))
	}
	input := inputs[0]
	if input.Domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", input.Domain)
	}
	if len(input.Headers) == 0 {
		t.Errorf("Expected headers, got none")
	}
	if len(input.Body) != 0 {
		t.Errorf("Expected empty body, got %d bytes", len(input.Body))
	}
	if serverHeader, ok := input.Headers["Server"]; !ok || serverHeader[0] != "headers-only-server" {
		t.Errorf("Expected Server header 'headers-only-server', got '%v'", serverHeader)
	}
}

func TestParseFFFBodyOnly(t *testing.T) {
	testRoot := "../../testdata/fff/body-only"
	inputsCh, err := fff.ParseFFF(testRoot, nil)
	if err != nil {
		t.Fatalf("ParseFFF failed: %v", err)
	}

	var inputs []model.OfflineInput
	for in := range inputsCh {
		inputs = append(inputs, in)
	}

	if len(inputs) != 1 {
		t.Fatalf("Expected 1 OfflineInput, got %d", len(inputs))
	}
	input := inputs[0]
	if input.Domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", input.Domain)
	}
	if len(input.Headers) != 0 {
		t.Errorf("Expected no headers, got %d", len(input.Headers))
	}
	if !strings.Contains(string(input.Body), "Body Only Content") {
		t.Errorf("Expected body to contain 'Body Only Content'")
	}
}

func TestParseFFFMultiDomain(t *testing.T) {
	testRoot := "../../testdata/fff/multi-domain"
	inputsCh, err := fff.ParseFFF(testRoot, nil)
	if err != nil {
		t.Fatalf("ParseFFF failed: %v", err)
	}

	var inputs []model.OfflineInput
	for in := range inputsCh {
		inputs = append(inputs, in)
	}

	if len(inputs) != 2 {
		t.Fatalf("Expected 2 OfflineInputs, got %d", len(inputs))
	}

	foundDomain1 := false
	foundDomain2 := false
	for _, input := range inputs {
		if input.Domain == "domain1.com" {
			foundDomain1 = true
			if input.URL != "https://domain1.com" {
				t.Errorf("Expected URL 'https://domain1.com', got '%s'", input.URL)
			}
			if serverHeader, ok := input.Headers["Server"]; !ok || serverHeader[0] != "domain1-server" {
				t.Errorf("Expected Server header 'domain1-server', got '%v'", serverHeader)
			}
			if !strings.Contains(string(input.Body), "Domain 1 Content") {
				t.Errorf("Expected body to contain 'Domain 1 Content'")
			}
		} else if input.Domain == "domain2.com" {
			foundDomain2 = true
			if input.URL != "https://domain2.com" {
				t.Errorf("Expected URL 'https://domain2.com', got '%s'", input.URL)
			}
			if serverHeader, ok := input.Headers["Server"]; !ok || serverHeader[0] != "domain2-server" {
				t.Errorf("Expected Server header 'domain2-server', got '%v'", serverHeader)
			}
			if !strings.Contains(string(input.Body), "Domain 2 Content") {
				t.Errorf("Expected body to contain 'Domain 2 Content'")
			}
		} else {
			t.Errorf("Unexpected domain: %s", input.Domain)
		}
	}

	if !foundDomain1 {
		t.Error("Did not find inputs for domain1.com")
	}
	if !foundDomain2 {
		t.Error("Did not find inputs for domain2.com")
	}
}

func TestDeriveURL(t *testing.T) {
	tests := []struct {
		name       string
		domainRoot string
		filePath   string
		domain     string
		expected   string
	}{
		{
			name:       "root file",
			domainRoot: "/tmp/fff/example.com",
			filePath:   "/tmp/fff/example.com/a94a8fe5ccb19ba61c4c0873d391e987982fbbd3.headers",
			domain:     "example.com",
			expected:   "https://example.com",
		},
		{
			name:       "nested file",
			domainRoot: "/tmp/fff/example.com",
			filePath:   "/tmp/fff/example.com/path/to/file/a94a8fe5ccb19ba61c4c0873d391e987982fbbd3.body",
			domain:     "example.com",
			expected:   "https://example.com/path/to/file",
		},
		{
			name:       "complex path",
			domainRoot: "/tmp/fff/www.divvyhomes.com",
			filePath:   "/tmp/fff/www.divvyhomes.com/_next/static/chunks/app/-layout-/privacy/page-c471709c92969dc3.js/a9b0e705b4b8c5b9da592350e29126ecf467a846.body",
			domain:     "www.divvyhomes.com",
			expected:   "https://www.divvyhomes.com/_next/static/chunks/app/-layout-/privacy/page-c471709c92969dc3.js",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := fff.DeriveURL(tt.domainRoot, tt.filePath, tt.domain)
			if actual != tt.expected {
				t.Errorf("deriveURL(%s, %s, %s) = %s; want %s", tt.domainRoot, tt.filePath, tt.domain, actual, tt.expected)
			}
		})
	}
}
