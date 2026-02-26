package katana_test

import (
	"strings"
	"testing"

	"github.com/Abhaythakor/hyperwapp/input/katana"
)

func TestParseKatanaFile(t *testing.T) {
	testFilePath := "../../testdata/katana/single-file/example.com_request.txt"
	fallbackDomain := "test.com"

	inputs, err := katana.ParseKatanaFile(testFilePath, fallbackDomain, nil)
	if err != nil {
		t.Fatalf("ParseKatanaFile failed: %v", err)
	}

	if len(inputs) != 1 {
		t.Fatalf("Expected 1 OfflineInput, got %d", len(inputs))
	}

	input := inputs[0]
	if input.Domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", input.Domain)
	}
	if input.URL != "https://example.com/test" { // Katana reconstructs URL from request line + Host header
		t.Errorf("Expected URL 'https://example.com/test', got '%s'", input.URL)
	}
	if len(input.Headers) == 0 {
		t.Errorf("Expected headers, got none")
	}
	if !strings.Contains(string(input.Body), "Katana Single File Test") {
		t.Errorf("Expected body to contain 'Katana Single File Test'")
	}

	if serverHeader, ok := input.Headers["Server"]; !ok || serverHeader[0] != "Apache" {
		t.Errorf("Expected Server header 'Apache', got '%v'", serverHeader)
	}
}
