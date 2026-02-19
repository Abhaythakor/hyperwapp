# PART 8

## Reference Go Implementation Plan

### Offline Parsers (fff, katana, raw HTTP, body-only)

---

## 8.1 Design Rules for Implementation

Before code, lock these rules:

1. **Parsers do not detect technologies**
2. **Parsers do not write output**
3. **Parsers only emit `OfflineInput`**
4. **Each parser is isolated**
5. **No parser knows about Wappalyzer**

This keeps the system testable and future-proof.

---

## 8.2 Core Types (Shared)

### `model/offline.go`

```go
package model

type OfflineInput struct {
	Domain  string
	URL     string
	Headers map[string][]string
	Body    []byte
}
```

---

## 8.3 Offline Parser Interface

All offline parsers must implement this.

```go
package input

import "hyperwapp/model"

type OfflineParser interface {
	Parse(path string) ([]model.OfflineInput, error)
}
```

This allows clean dispatch based on detected format.

---

## 8.4 Dispatcher (Glue Code)

### `input/offline.go`

```go
func ParseOffline(path string) ([]model.OfflineInput, error) {
	format := DetectOfflineFormat(path)

	switch format {
	case FormatFFF:
		return ParseFFF(path)
	case FormatKatanaDir:
		return ParseKatanaDir(path)
	case FormatKatanaFile:
		return ParseKatanaFile(path)
	case FormatRawHTTP:
		return ParseRawHTTP(path)
	case FormatBodyOnly:
		return ParseBodyOnly(path)
	default:
		return nil, fmt.Errorf("unsupported offline format")
	}
}
```

---

# 8.5 fff PARSER (REFERENCE IMPLEMENTATION)

### File: `input/fff.go`

---

## 8.5.1 High-Level fff Algorithm

```
root dir
 ├─ domain dir
 │   ├─ recursive walk
 │   ├─ find *.headers / *.body
 │   ├─ group by hash
 │   ├─ derive URL path
 │   └─ emit OfflineInput
```

---

## 8.5.2 fff Parser Skeleton

```go
func ParseFFF(root string) ([]model.OfflineInput, error) {
	var results []model.OfflineInput

	domainDirs, _ := os.ReadDir(root)
	for _, d := range domainDirs {
		if !d.IsDir() {
			continue
		}
		domain := d.Name()
		domainPath := filepath.Join(root, domain)

		inputs, err := parseFFFDomain(domainPath, domain)
		if err != nil {
			log.Warn(err)
			continue
		}
		results = append(results, inputs...)
	}
	return results, nil
}
```

---

## 8.5.3 Domain-Level Parsing

```go
func parseFFFDomain(path, domain string) ([]model.OfflineInput, error) {
	filesByHash := map[string]map[string]string{}

	filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if strings.HasSuffix(p, ".headers") || strings.HasSuffix(p, ".body") {
			hash := extractHash(p)
			if _, ok := filesByHash[hash]; !ok {
				filesByHash[hash] = map[string]string{}
			}
			if strings.HasSuffix(p, ".headers") {
				filesByHash[hash]["headers"] = p
			}
			if strings.HasSuffix(p, ".body") {
				filesByHash[hash]["body"] = p
			}
		}
		return nil
	})

	return buildFFFInputs(filesByHash, path, domain), nil
}
```

---

## 8.5.4 URL Reconstruction Helper

```go
func deriveURL(domainRoot, filePath, domain string) string {
	rel, _ := filepath.Rel(domainRoot, filepath.Dir(filePath))
	if rel == "." {
		return "https://" + domain
	}
	return "https://" + domain + "/" + filepath.ToSlash(rel)
}
```

---

## 8.5.5 Building OfflineInput Objects

```go
func buildFFFInputs(groups map[string]map[string]string, root, domain string) []model.OfflineInput {
	var inputs []model.OfflineInput

	for _, files := range groups {
		headers := map[string][]string{}
		body := []byte{}

		if h, ok := files["headers"]; ok {
			headers = parseHeadersFile(h)
		}
		if b, ok := files["body"]; ok {
			body, _ = os.ReadFile(b)
		}

		url := deriveURL(root, firstFile(files), domain)

		inputs = append(inputs, model.OfflineInput{
			Domain:  domain,
			URL:     url,
			Headers: headers,
			Body:    body,
		})
	}
	return inputs
}
```

---

# 8.6 KATANA PARSER (REFERENCE)

### File: `input/katana.go`

---

## 8.6.1 Katana Directory Parser

```go
func ParseKatanaDir(root string) ([]model.OfflineInput, error) {
	var results []model.OfflineInput

	dirs, _ := os.ReadDir(root)
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		domain := d.Name()
		files, _ := os.ReadDir(filepath.Join(root, domain))

		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".txt") {
				path := filepath.Join(root, domain, f.Name())
				input, err := parseKatanaFile(path, domain)
				if err == nil {
					results = append(results, input)
				}
			}
		}
	}
	return results, nil
}
```

---

## 8.6.2 Katana File Parser

```go
func ParseKatanaFile(path string) ([]model.OfflineInput, error) {
	input, err := parseKatanaFile(path, "")
	if err != nil {
		return nil, err
	}
	return []model.OfflineInput{input}, nil
}
```

---

## 8.6.3 Katana File Core Logic

```go
func parseKatanaFile(path, fallbackDomain string) (model.OfflineInput, error) {
	data, _ := os.ReadFile(path)
	parts := splitRequestResponse(data)

	headers := parseResponseHeaders(parts.ResponseHeaders)
	body := parts.Body
	domain := extractHost(headers, fallbackDomain)
	url := reconstructURL(parts.RequestLine, headers)

	return model.OfflineInput{
		Domain:  domain,
		URL:     url,
		Headers: headers,
		Body:    body,
	}, nil
}
```

---

# 8.7 RAW HTTP PARSER

### File: `input/raw.go`

```go
func ParseRawHTTP(path string) ([]model.OfflineInput, error) {
	data, _ := os.ReadFile(path)
	responses := splitHTTPResponses(data)

	var inputs []model.OfflineInput
	for _, r := range responses {
		headers := parseHeaders(r.Headers)
		body := r.Body
		domain := extractHost(headers, "unknown")

		inputs = append(inputs, model.OfflineInput{
			Domain:  domain,
			URL:     "",
			Headers: headers,
			Body:    body,
		})
	}
	return inputs, nil
}
```

---

# 8.8 BODY-ONLY PARSER

### File: `input/body.go`

```go
func ParseBodyOnly(path string) ([]model.OfflineInput, error) {
	body, _ := os.ReadFile(path)

	return []model.OfflineInput{
		{
			Domain:  inferDomain(path),
			URL:     "",
			Headers: map[string][]string{},
			Body:    body,
		},
	}, nil
}
```

---

## 8.9 Error Handling Rules (Implementation)

- Parsers **never panic**
- Partial inputs are allowed
- Errors are logged, not fatal
- Empty results are skipped

---

## 8.10 Testing Strategy (Preview)

Each parser must have:

- Valid fixture
- Missing headers
- Missing body
- Corrupt file
- Large input

(Expanded fully in Part 9.)

---

## 8.11 What PART 8 Gives You

After this part, you have:

- Concrete parser boundaries
- Code-shaped logic
- Clean separation of concerns
- Zero ambiguity for implementation
