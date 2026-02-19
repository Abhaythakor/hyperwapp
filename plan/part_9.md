# PART 9

## Test Strategy, Fixtures & Validation

### Offline Inputs (fff, katana, raw HTTP, body-only)

---

## 9.1 Testing Philosophy

Offline parsing is the **highest-risk surface** of the tool.

Therefore:

- Tests must be **format-specific**
- Tests must validate **normalization**, not detection
- Tests must be **file-backed**, not mocked
- Tests must survive malformed input
- Tests must assert **schema correctness**

We test parsing.
We do **not** test Wappalyzer here.

---

## 9.2 What Tests Must Guarantee

For every offline format:

1. No panic
2. Correct format detection
3. Correct domain extraction
4. Correct URL reconstruction (when possible)
5. Headers parsed correctly
6. Body passed through unchanged
7. `OfflineInput` schema always valid

---

## 9.3 Test Directory Layout

Add this to the repository:

```
testdata/
├── fff/
│   ├── simple/
│   ├── nested/
│   ├── headers-only/
│   └── body-only/
│
├── katana/
│   ├── directory/
│   ├── single-file/
│   └── malformed/
│
├── raw-http/
│   ├── single-response.txt
│   ├── multiple-responses.txt
│   └── malformed.txt
│
├── body-only/
│   ├── page.html
│   ├── script.js
│   └── style.css
│
└── edge-cases/
    ├── empty-file.txt
    ├── binary-body.bin
    └── missing-host.headers
```

All tests reference files from here.

---

## 9.4 Test Coverage Matrix

| Format      | Test Case       | Expected         |
| ----------- | --------------- | ---------------- |
| fff         | headers + body  | 1 OfflineInput   |
| fff         | headers only    | body empty       |
| fff         | body only       | headers empty    |
| fff         | deep paths      | correct URL      |
| fff         | multi-domain    | domains isolated |
| katana dir  | valid responses | N inputs         |
| katana file | single file     | 1 input          |
| katana      | malformed       | skipped, warn    |
| raw HTTP    | single response | 1 input          |
| raw HTTP    | multiple        | N inputs         |
| body-only   | html/js/css     | body only        |

---

## 9.5 Format Detection Tests

### File: `input/detect_test.go`

```go
func TestDetectFFF(t *testing.T) {
	format := DetectOfflineFormat("testdata/fff/simple")
	if format != FormatFFF {
		t.Fatalf("expected fff, got %s", format)
	}
}
```

Repeat for:

- katana directory
- katana file
- raw HTTP
- body-only

---

## 9.6 fff Parser Tests

### File: `input/fff_test.go`

#### Basic fff test

```go
func TestFFFBasic(t *testing.T) {
	inputs, err := ParseFFF("testdata/fff/simple")
	if err != nil {
		t.Fatal(err)
	}

	if len(inputs) != 1 {
		t.Fatalf("expected 1 input, got %d", len(inputs))
	}

	in := inputs[0]
	if in.Domain != "example.com" {
		t.Fatalf("wrong domain")
	}
	if in.URL != "https://example.com/privacy" {
		t.Fatalf("wrong URL")
	}
}
```

---

#### Headers-only fff test

```go
func TestFFFHeadersOnly(t *testing.T) {
	inputs, _ := ParseFFF("testdata/fff/headers-only")

	if len(inputs[0].Body) != 0 {
		t.Fatalf("body should be empty")
	}
}
```

---

#### Deep path reconstruction

```go
func TestFFFDeepPath(t *testing.T) {
	inputs, _ := ParseFFF("testdata/fff/nested")

	if !strings.Contains(inputs[0].URL, "/_next/static/") {
		t.Fatalf("URL path incorrect")
	}
}
```

---

## 9.7 Katana Parser Tests

### File: `input/katana_test.go`

#### Katana directory test

```go
func TestKatanaDir(t *testing.T) {
	inputs, err := ParseKatanaDir("testdata/katana/directory")
	if err != nil {
		t.Fatal(err)
	}

	if len(inputs) == 0 {
		t.Fatalf("expected inputs")
	}
}
```

---

#### Katana malformed file test

```go
func TestKatanaMalformed(t *testing.T) {
	inputs, err := ParseKatanaFile("testdata/katana/malformed/bad.txt")
	if err != nil {
		t.Fatal(err)
	}

	if len(inputs) != 0 {
		t.Fatalf("malformed input should be skipped")
	}
}
```

---

## 9.8 Raw HTTP Parser Tests

### File: `input/raw_test.go`

```go
func TestRawHTTPMultipleResponses(t *testing.T) {
	inputs, err := ParseRawHTTP("testdata/raw-http/multiple-responses.txt")
	if err != nil {
		t.Fatal(err)
	}

	if len(inputs) < 2 {
		t.Fatalf("expected multiple responses")
	}
}
```

---

## 9.9 Body-Only Tests

### File: `input/body_test.go`

```go
func TestBodyOnlyHTML(t *testing.T) {
	inputs, err := ParseBodyOnly("testdata/body-only/page.html")
	if err != nil {
		t.Fatal(err)
	}

	if len(inputs[0].Headers) != 0 {
		t.Fatalf("headers should be empty")
	}
}
```

---

## 9.10 Schema Validation Tests

### File: `model/schema_test.go`

Ensure every emitted input satisfies schema rules.

```go
func TestOfflineInputSchema(t *testing.T) {
	inputs, _ := ParseFFF("testdata/fff/simple")

	for _, in := range inputs {
		if in.Domain == "" {
			t.Fatalf("domain must not be empty")
		}
		if in.Headers == nil {
			t.Fatalf("headers map must exist")
		}
	}
}
```

---

## 9.11 Negative & Edge Case Tests

Include tests for:

- Empty files
- Binary bodies
- Missing headers
- Missing body
- Permission denied files

All must **fail gracefully**.

---

## 9.12 Test Execution Strategy

- Tests run via:

  ```
  go test ./...
  ```

- No network calls
- No Wappalyzer dependency
- No flaky timing

---

## 9.13 CI Enforcement (Preview)

Later (optional):

- Block merge if:
  - Any parser test fails
  - Format detection misclassifies input
  - Panic occurs

---

## 9.14 What PART 9 Guarantees

After implementing this part:

- Offline parsing is safe
- Format detection is stable
- fff and katana regressions are caught early
- New formats can be added confidently
