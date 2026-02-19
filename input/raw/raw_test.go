package raw_test

import (
	"strings"
	"testing"

	"hyperwapp/input/raw"
	"hyperwapp/model"
)

func TestParseRawHTTPSingleResponse(t *testing.T) {
	testFilePath := "../../testdata/raw-http/single-response.txt"

	inputsCh, err := raw.ParseRawHTTP(testFilePath)
	if err != nil {
		t.Fatalf("ParseRawHTTP failed: %v", err)
	}

	var inputs []model.OfflineInput
	for in := range inputsCh {
		inputs = append(inputs, in)
	}

	if len(inputs) != 1 {
		t.Fatalf("Expected 1 OfflineInput, got %d", len(inputs))
	}

	input := inputs[0]
	if input.Domain != "unknown" { // Domain cannot be inferred from raw response without Host header
		t.Errorf("Expected domain 'unknown', got '%s'", input.Domain)
	}
	if input.URL != "" {
		t.Errorf("Expected empty URL, got '%s'", input.URL)
	}
	if len(input.Headers) == 0 {
		t.Errorf("Expected headers, got none")
	}
	if !strings.Contains(string(input.Body), "Single Raw HTTP Response") {
		t.Errorf("Expected body to contain 'Single Raw HTTP Response'")
	}

	if serverHeader, ok := input.Headers["Server"]; !ok || serverHeader[0] != "Apache/2.4.1 (Unix)" {
		t.Errorf("Expected Server header 'Apache/2.4.1 (Unix)', got '%v'", serverHeader)
	}
}

func TestParseRawHTTPMultipleResponses(t *testing.T) {
	testFilePath := "../../testdata/raw-http/multiple-responses.txt"

	inputsCh, err := raw.ParseRawHTTP(testFilePath)
	if err != nil {
		t.Fatalf("ParseRawHTTP failed: %v", err)
	}

	var inputs []model.OfflineInput
	for in := range inputsCh {
		inputs = append(inputs, in)
	}

	if len(inputs) != 2 {
		t.Fatalf("Expected 2 OfflineInputs, got %d", len(inputs))
	}

	// Test first response
	input1 := inputs[0]
	if input1.Domain != "unknown" {
		t.Errorf("Expected domain 'unknown', got '%s'", input1.Domain)
	}
	if !strings.Contains(string(input1.Body), "Multiple Raw HTTP Response - First") {
		t.Errorf("Expected body to contain 'Multiple Raw HTTP Response - First'")
	}

	// Test second response
	input2 := inputs[1]
	if input2.Domain != "unknown" {
		t.Errorf("Expected domain 'unknown', got '%s'", input2.Domain)
	}
	if !strings.Contains(string(input2.Body), "Not Found") {
		t.Errorf("Expected body to contain 'Not Found'")
	}
}

func TestParseRawHTTPMalformed(t *testing.T) {
	testFilePath := "../../testdata/raw-http/malformed.txt"

	inputsCh, err := raw.ParseRawHTTP(testFilePath)
	if err != nil {
		t.Fatalf("ParseRawHTTP should not return an error for malformed content, got: %v", err)
	}

	var inputs []model.OfflineInput
	for in := range inputsCh {
		inputs = append(inputs, in)
	}

	if len(inputs) != 0 {
		t.Errorf("Expected 0 OfflineInputs for malformed, got %d", len(inputs))
	}
}
