# PART 6

## Offline Input: Multi-Format Support (FINAL, AUTHORITATIVE)

This part defines **exactly how offline input is detected, parsed, normalized, and fed into WappalyzerGo** for **katana**, **fff**, **raw HTTP**, and **body-only** inputs.

---

## 6.1 Core Philosophy (Non-Negotiable)

Offline mode exists to **reconstruct HTTP context**, not to re-implement detection.

Rules:

1. Offline parsing is **format-aware**
2. Detection is **format-agnostic**
3. Every offline input must normalize into the same structure
4. CSV / JSON output is identical to online mode
5. Best-effort parsing, never brittle

---

## 6.2 Canonical Offline Normalization Output

Every offline parser must emit **this exact structure**:

```go
OfflineInput {
  Domain  string
  URL     string        // optional
  Headers map[string][]string
  Body    []byte
}
```

Once this structure exists, **everything else is identical to online mode**:

```
OfflineInput → Wappalyzer → Detection Records → Export
```

---

## 6.3 Supported Offline Input Types

The tool MUST support the following offline formats:

1. **fff directory output** (headers + body split)
2. **katana response directories**
3. **Single katana response files**
4. **Raw HTTP response dumps**
5. **Body-only files (HTML / JS / CSS / JSON)**

---

## 6.4 Offline Format Auto-Detection Order

Format detection is deterministic and ordered:

1. **fff directory structure**
2. **katana directory structure**
3. **single katana response file**
4. **raw HTTP dump**
5. **body-only file**

First positive match wins.

This order matters because **fff and katana both use directories**, but semantics differ.

---

# 6.5 fff OFFLINE FORMAT (PRIMARY)

## 6.5.1 fff Structure Characteristics

fff output is **directory-based**, not stream-based.

Key invariants:

- Headers and body are stored in **separate files**
- Files share a **common hash prefix**
- Directory structure represents the **URL path**
- Top-level directory represents the **domain**

Example pattern (simplified):

```
fff/
└── example.com
    ├── path
    │   ├── <hash>.headers
    │   └── <hash>.body
    └── _next/static/...
```

---

## 6.5.2 fff Format Detection Rules

A directory is considered **fff output** if:

- Files end in `.headers` and/or `.body`
- Matching hash prefixes exist
- Headers and body are **separate files**

No heuristics. This is exact.

---

## 6.5.3 fff Traversal Strategy

### Step 1: Domain Root Detection

Each **first-level directory** under the input root is treated as a domain:

```
fff/example.com → domain = example.com
```

---

### Step 2: Recursive Walk

- Walk all subdirectories
- Ignore directories with no `.headers` or `.body` files
- Group files by hash prefix

Valid groups:

- headers + body
- headers only
- body only

---

## 6.5.4 URL Reconstruction (Critical)

### URL path derivation

The URL path is derived from the directory path **relative to the domain root**.

Examples:

```
fff/example.com/privacy/
→ /privacy

fff/example.com/_next/static/chunks/app/
→ /_next/static/chunks/app
```

---

### URL construction rule

```
https://{domain}{path}
```

If the path cannot be determined:

- URL may be empty
- Domain MUST still be populated

---

## 6.5.5 Parsing `.headers` Files (fff)

Rules:

- Headers files contain **response headers**
- Status line may exist
- One header per line

Parsing rules:

- Ignore status line (`HTTP/1.x`)
- Split on first `:`
- Normalize header names
- Preserve multiple values

Result example:

```go
headers["Content-Type"] = []string{"text/html; charset=utf-8"}
headers["X-Powered-By"] = []string{"Express"}
```

---

## 6.5.6 Parsing `.body` Files (fff)

Rules:

- Raw response body
- No decoding
- No guessing
- No transformations

The file contents are passed **as-is** to Wappalyzer.

---

## 6.5.7 fff Normalized Output

Each valid hash group produces **one OfflineInput**:

```go
OfflineInput {
  Domain:  "example.com",
  URL:     "https://example.com/privacy",
  Headers: headers,
  Body:    body,
}
```

---

## 6.5.8 fff Flag Interaction

| Flag            | Behavior                 |
| --------------- | ------------------------ |
| `-headers-only` | Load headers, empty body |
| `-body-only`    | Load body, empty headers |
| `-auto`         | Load both                |

---

## 6.5.9 fff Edge Cases (Handled)

| Case               | Result                |
| ------------------ | --------------------- |
| Missing `.body`    | Header-only detection |
| Missing `.headers` | Body-only detection   |
| Binary body        | Passed as-is          |
| Deep JS assets     | Valid URLs            |
| Multiple domains   | Fully isolated        |

---

# 6.6 KATANA OFFLINE FORMAT

## 6.6.1 Katana Directory Structure

Katana response directories typically look like:

```
katana_response/
├── index.txt
└── example.com
    ├── <hash>.txt
    ├── <hash>.txt
```

Each `.txt` file represents **one HTTP transaction**.

---

## 6.6.2 Katana File Structure

Katana response files contain:

1. Request line + headers
2. Blank line
3. Response status line
4. Response headers
5. Blank line
6. Response body

---

## 6.6.3 Katana Parsing Rules

- Ignore request headers
- Parse **response headers only**
- Body = everything after response header block

---

## 6.6.4 Katana Domain & URL Extraction

- Domain:
  - From parent directory name

- URL:
  - Reconstructed from request path + Host header
  - If unavailable, URL may be empty

---

## 6.6.5 Katana Normalized Output

Each file produces:

```go
OfflineInput {
  Domain:  "example.com",
  URL:     "https://example.com/path",
  Headers: headers,
  Body:    body,
}
```

---

# 6.7 RAW HTTP RESPONSE FORMAT

## 6.7.1 Raw HTTP Detection Rules

A file is considered raw HTTP if it contains:

- `HTTP/1.0` or `HTTP/1.1`
- At least one `Header: Value` line

---

## 6.7.2 Raw HTTP Parsing Rules

- Prefer response headers
- Ignore request headers
- Split multiple responses if present
- URL may be empty
- Domain derived from `Host` header if available

---

# 6.8 BODY-ONLY FILES

## 6.8.1 Supported Body-Only Inputs

- `.html`
- `.js`
- `.css`
- `.json`
- Minified assets

---

## 6.8.2 Body-Only Normalization

```go
OfflineInput {
  Domain:  inferred or "unknown",
  URL:     "",
  Headers: empty map,
  Body:    full file contents,
}
```

Wappalyzer still detects JS frameworks reliably.

---

# 6.9 Unified Offline → Detection Flow

```
Offline input
 ↓
Format detection
 ↓
Format-specific parser
 ↓
OfflineInput normalization
 ↓
Wappalyzer fingerprint
 ↓
Detection records
 ↓
CSV / JSON / CLI
```

---

## 6.10 Error Handling (Offline)

Offline mode is **best-effort**, never brittle.

| Condition       | Behavior   |
| --------------- | ---------- |
| Invalid file    | Skip, warn |
| Partial headers | Continue   |
| Missing body    | Continue   |
| Parse failure   | Warn       |
| Empty result    | Skip       |

Fatal errors only occur if **explicitly requested**.

---

## 6.11 Guarantees Provided by PART 6

After this part, the system:

- Fully supports **fff**
- Fully supports **katana**
- Handles raw HTTP dumps
- Handles body-only artifacts
- Feeds Wappalyzer correctly in all cases
- Produces identical CSV/JSON schemas
