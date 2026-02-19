package body_test

import (
	"strings"
	"testing"

	"github.com/Abhaythakor/hyperwapp/input/body"
	"github.com/Abhaythakor/hyperwapp/model"
)

func TestParseBodyOnlyHTML(t *testing.T) {
	testFilePath := "../../testdata/body-only/page.html"

	inputsCh, err := body.ParseBodyOnly(testFilePath)
	if err != nil {
		t.Fatalf("ParseBodyOnly failed: %v", err)
	}

	var inputs []model.OfflineInput
	for in := range inputsCh {
		inputs = append(inputs, in)
	}

	if len(inputs) != 1 {
		t.Fatalf("Expected 1 OfflineInput, got %d", len(inputs))
	}

	input := inputs[0]
	if input.Domain != "page" { // Inferred from filename without extension
		t.Errorf("Expected domain 'page', got '%s'", input.Domain)
	}
	if input.URL != "" {
		t.Errorf("Expected empty URL, got '%s'", input.URL)
	}
	if len(input.Headers) != 0 {
		t.Errorf("Expected no headers, got %d", len(input.Headers))
	}
	if !strings.Contains(string(input.Body), "Body Only HTML Test") {
		t.Errorf("Expected body to contain 'Body Only HTML Test'")
	}
}

func TestParseBodyOnlyJS(t *testing.T) {
	testFilePath := "../../testdata/body-only/script.js"

	inputsCh, err := body.ParseBodyOnly(testFilePath)
	if err != nil {
		t.Fatalf("ParseBodyOnly failed: %v", err)
	}

	var inputs []model.OfflineInput
	for in := range inputsCh {
		inputs = append(inputs, in)
	}

	if len(inputs) != 1 {
		t.Fatalf("Expected 1 OfflineInput, got %d", len(inputs))
	}

	input := inputs[0]
	if input.Domain != "script" { // Inferred from filename without extension
		t.Errorf("Expected domain 'script', got '%s'", input.Domain)
	}
	if !strings.Contains(string(input.Body), "Hello Vue!") {
		t.Errorf("Expected body to contain 'Hello Vue!'")
	}
}

func TestParseBodyOnlyCSS(t *testing.T) {
	testFilePath := "../../testdata/body-only/style.css"

	inputsCh, err := body.ParseBodyOnly(testFilePath)
	if err != nil {
		t.Fatalf("ParseBodyOnly failed: %v", err)
	}

	var inputs []model.OfflineInput
	for in := range inputsCh {
		inputs = append(inputs, in)
	}

	if len(inputs) != 1 {
		t.Fatalf("Expected 1 OfflineInput, got %d", len(inputs))
	}

	input := inputs[0]
	if input.Domain != "style" { // Inferred from filename without extension
		t.Errorf("Expected domain 'style', got '%s'", input.Domain)
	}
	if !strings.Contains(string(input.Body), "background-color: #f0f0f0;") {
		t.Errorf("Expected body to contain 'background-color: #f0f0f0;'")
	}
}
