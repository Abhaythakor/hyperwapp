# PART 7

## Offline Auto-Detection Logic & Parsing Pseudocode (IMPLEMENTATION GUIDE)

This part defines **exact decision logic**, **parsing flow**, and **pseudocode** for detecting and handling offline inputs reliably.

---

## 7.1 Goals of Auto-Detection

Offline auto-detection must:

- Correctly identify format with **high confidence**
- Avoid false positives
- Prefer **structure over content heuristics**
- Be deterministic and debuggable
- Fail safely (best-effort)

---

## 7.2 Offline Detection Entry Point

### Function signature

```go
func DetectOfflineFormat(path string) OfflineFormat
```

### Supported formats (enum)

```go
type OfflineFormat string

const (
  FormatFFF        OfflineFormat = "fff"
  FormatKatanaDir  OfflineFormat = "katana-dir"
  FormatKatanaFile OfflineFormat = "katana-file"
  FormatRawHTTP    OfflineFormat = "raw-http"
  FormatBodyOnly   OfflineFormat = "body-only"
)
```

---

## 7.3 High-Level Detection Algorithm

### Decision tree (authoritative)

```
Is directory?
 ├─ Yes → Is fff structure?
 │        ├─ Yes → fff
 │        └─ No → Is katana directory?
 │                 ├─ Yes → katana-dir
 │                 └─ No → error / unsupported
 └─ No  → Is katana response file?
          ├─ Yes → katana-file
          └─ No → Is raw HTTP?
                   ├─ Yes → raw-http
                   └─ No → body-only
```

---

## 7.4 fff Directory Detection (Exact)

### Detection conditions

A directory is **fff** if:

- Contains files ending in `.headers` or `.body`
- At least one matching hash prefix exists
- Headers and body are separate files

### Pseudocode

```go
func isFFFDirectory(path string) bool {
  foundHeaders := false
  foundBody := false

  walk(path, func(file string) {
    if strings.HasSuffix(file, ".headers") {
      foundHeaders = true
    }
    if strings.HasSuffix(file, ".body") {
      foundBody = true
    }
  })

  return foundHeaders && foundBody
}
```

This is strict and safe.

---

## 7.5 Katana Directory Detection

### Detection conditions

A directory is **katana** if:

- Contains `index.txt` OR
- Contains subdirectories with `.txt` response files
- Files do **not** use `.headers` / `.body` split

### Pseudocode

```go
func isKatanaDirectory(path string) bool {
  if exists(path + "/index.txt") {
    return true
  }

  foundTxt := false
  walk(path, func(file string) {
    if strings.HasSuffix(file, ".txt") {
      foundTxt = true
    }
  })

  return foundTxt
}
```

---

## 7.6 Katana Single File Detection

### Detection conditions

A file is **katana response file** if:

- Contains both:
  - Request line (`GET /`)
  - Response line (`HTTP/1.`)

- Contains **two header blocks**

### Pseudocode

```go
func isKatanaFile(file string) bool {
  data := readFirstKB(file)

  return contains(data, "GET ") &&
         contains(data, "HTTP/1.")
}
```

---

## 7.7 Raw HTTP Detection

### Detection conditions

A file is **raw HTTP** if:

- Contains `HTTP/1.0` or `HTTP/1.1`
- Contains header lines (`Key: Value`)
- Does NOT contain request + response pair

### Pseudocode

```go
func isRawHTTP(file string) bool {
  data := readFirstKB(file)

  if !contains(data, "HTTP/1.") {
    return false
  }

  if contains(data, "GET ") {
    return false // katana-style
  }

  return containsHeaderLine(data)
}
```

---

## 7.8 Body-Only Detection

This is the **fallback format**.

A file is treated as **body-only** if:

- None of the above formats match
- File is readable
- File size > 0

No content guessing is required.

---

## 7.9 Offline Format Detection (Final Pseudocode)

```go
func DetectOfflineFormat(path string) OfflineFormat {
  if isDirectory(path) {
    if isFFFDirectory(path) {
      return FormatFFF
    }
    if isKatanaDirectory(path) {
      return FormatKatanaDir
    }
  } else {
    if isKatanaFile(path) {
      return FormatKatanaFile
    }
    if isRawHTTP(path) {
      return FormatRawHTTP
    }
  }

  return FormatBodyOnly
}
```

This function is **pure** and **side-effect free**.

---

## 7.10 Parser Dispatch Logic

Once detected:

```go
switch format {
case FormatFFF:
  parseFFF(path)
case FormatKatanaDir:
  parseKatanaDirectory(path)
case FormatKatanaFile:
  parseKatanaFile(path)
case FormatRawHTTP:
  parseRawHTTP(path)
case FormatBodyOnly:
  parseBodyOnly(path)
}
```

Each parser emits **OfflineInput objects**.

---

## 7.11 Offline Parsing Contract (Critical)

Every parser MUST:

- Emit zero or more `OfflineInput`
- Never panic on malformed data
- Never call Wappalyzer directly
- Never write output
- Never aggregate

Parsing and detection are strictly separated.

---

## 7.12 Logging & Debugging Rules

When `-verbose` is enabled:

- Log detected format
- Log file count
- Log skipped files
- Log normalization results (counts only)

Example:

```
[offline] detected format: fff
[offline] domains: 3
[offline] inputs normalized: 428
```

---

## 7.13 Performance Considerations

- Directory walking must be streaming
- Never load entire trees into memory
- Read headers fully
- Read body lazily when possible
- Emit OfflineInput incrementally

This allows very large fff / katana datasets.

---

## 7.14 Guarantees of PART 7

With this part implemented:

- Offline format detection is deterministic
- fff and katana never conflict
- Raw HTTP never misclassified
- Body-only fallback is safe
- Parsing is debuggable and maintainable
