# TECHSCAN – FULL UPDATED DETAILED PLAN

## (WappalyzerGo-Based Architecture)

---

# PART 1

## Goals, Principles, Architecture, CLI

---

## 1. Tool Overview

**Name:** `hyperwapp`
**Type:** CLI reconnaissance utility
**Core Engine:** `github.com/projectdiscovery/wappalyzergo`

### What it does

- Detects web technologies using Wappalyzer fingerprints
- Works with:
  - Single URL
  - URL list
  - Offline files (headers, body, or mixed)

- Exports **lossless structured data**
- Aggregates by domain when requested

---

## 2. Design Principles (Non-Negotiable)

1. Detection is **delegated**, not reimplemented
2. CSV and JSON are **authoritative outputs**
3. Every detection is a **fact**, not a summary
4. Offline and online modes produce identical schemas
5. Aggregation never destroys raw detections
6. CLI output is presentation only

---

## 3. High-Level Architecture

```
Input
 ├─ Online HTTP
 └─ Offline File
        ↓
Headers + Body
        ↓
WappalyzerGo Engine
        ↓
Detection Records (hyperwapp schema)
        ↓
Aggregation (optional)
        ↓
CSV / JSON / CLI / TXT / MD
```

---

## 4. External Dependency

### Required library

```bash
go get github.com/projectdiscovery/wappalyzergo
```

### Fingerprint updates (user-managed)

```bash
go install -v github.com/projectdiscovery/wappalyzergo/cmd/update-fingerprints@latest
update-fingerprints
```

**Important:**
Fingerprints are **not bundled**. They are updated independently.

---

## 5. CLI Interface

### Basic syntax

```
hyperwapp [input] [flags]
```

### Input examples

```
hyperwapp https://example.com
hyperwapp urls.txt
cat urls.txt | hyperwapp
hyperwapp -offline scan.txt
```

---

## 6. Flags (Final)

### Input mode flags

| Flag            | Description              |
| --------------- | ------------------------ |
| `-offline`      | Offline parsing mode     |
| `-headers-only` | Use headers only         |
| `-body-only`    | Use body only            |
| `-auto`         | Headers + body (default) |

---

### Output behavior flags

| Flag      | Description         |
| --------- | ------------------- |
| `-all`    | Per-URL output      |
| `-domain` | Aggregate by domain |

---

### Export flags

| Flag      | Description        |
| --------- | ------------------ |
| `-o`      | Output file        |
| `-format` | csv, json, txt, md |

---

### Performance flags

| Flag       | Description        |
| ---------- | ------------------ |
| `-threads` | Concurrent workers |
| `-timeout` | HTTP timeout       |

---

## 7. Input Resolution Logic

Priority order:

1. stdin
2. positional argument
3. file path

Rules:

- `-offline` disables HTTP
- Only one offline file allowed
- Mixed offline/online is forbidden
- Invalid lines are skipped with warning

---

## 8. Target Normalization

For each input line:

- Trim whitespace
- Validate URL (online mode)
- Extract:
  - domain
  - full URL

- Deduplicate targets

---

## 9. Online Mode Flow

```
URL
 ↓
HTTP GET
 ↓
Capture headers
 ↓
Read body
 ↓
Send to Wappalyzer engine
 ↓
Detection records
```

---

## 10. Offline Mode Flow

Offline mode exists to **feed Wappalyzer**, not replace it.

```
File / stdin
 ↓
Parse headers
 ↓
Parse body
 ↓
Send to Wappalyzer engine
 ↓
Detection records
```

Offline parsing is responsible only for **reconstructing headers and body**.

---

## 11. Detection Responsibility Split

### WappalyzerGo does

- Identify technologies
- Maintain fingerprint rules
- Analyze headers and body

### Techscan does

- Provide headers and body
- Add domain and URL context
- Add timestamps
- Export structured data
- Aggregate results
