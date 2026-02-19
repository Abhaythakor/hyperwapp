# PART 2

## Canonical Data Model & Wappalyzer Mapping

---

## 12. Canonical Detection Model (Authoritative)

This model is the **single source of truth**.
Everything else adapts to it.

```go
Detection {
  Domain     string        // example.com
  URL        string        // https://example.com
  Technology string        // React, Cloudflare, Apache
  Source     string        // wappalyzer
  Path       string        // fingerprint
  Evidence   string        // wappalyzergo
  Confidence string        // high
  Timestamp  time.Time    // RFC3339
}
```

Rules:

- One detection per technology per scan
- No deduplication across scans
- Timestamp always present
- URL may be empty in domain mode or offline mode

---

## 13. Supporting Models

### Target

```go
Target {
  URL    string
  Domain string
}
```

### Scan Metadata

```go
Meta {
  Tool        string
  Version     string
  GeneratedAt time.Time
  Mode        string        // all | domain
  InputType   string        // online | offline
}
```

---

## 14. Wappalyzer Engine Wrapper

### Why wrap it

- Isolate external dependency
- Avoid leaking its types
- Make hybrid detection easy later

---

### Engine initialization

```go
engine, err := detect.NewWappalyzerEngine()
```

- Loads fingerprints automatically
- Fails fast if fingerprints missing

---

## 15. Fingerprinting Execution

### Inputs

- `http.Header`
- `[]byte` body

### Wappalyzer output

```go
map[string]struct{}
```

Example:

```
map[Cloudflare:{} React:{} Drupal:{}]
```

---

## 16. Mapping Output to Detection Records

Wappalyzer does **not expose raw match details**, so mapping is explicit and honest.

| Detection Field | Value              |
| --------------- | ------------------ |
| `Technology`    | map key            |
| `Source`        | `wappalyzer`       |
| `Path`          | `fingerprint`      |
| `Evidence`      | `wappalyzergo`     |
| `Confidence`    | `high`             |
| `Timestamp`     | `time.Now().UTC()` |

---

### Mapping logic (conceptual)

```go
for tech := range fingerprints {
  emit Detection{
    Domain: domain,
    URL: url,
    Technology: tech,
    Source: "wappalyzer",
    Path: "fingerprint",
    Evidence: "wappalyzergo",
    Confidence: "high",
    Timestamp: now,
  }
}
```

---

## 17. Source Refinement (Optional Enhancement)

If flags are used:

- `-headers-only`
- `-body-only`

Then:

| Flag         | Source value      |
| ------------ | ----------------- |
| headers-only | wappalyzer-header |
| body-only    | wappalyzer-body   |
| auto         | wappalyzer        |

This is cosmetic but useful.

---

## 18. Confidence Strategy

Wappalyzer detections are considered:

- `high` confidence by default
- Offline body-only can optionally downgrade to `medium`

This is configurable, not hardcoded.

---

## 19. Multi-URL Scan Behavior

- Same technology on different URLs → separate Detection records
- Same URL scanned twice → two timestamps
- No global deduplication

---

## 20. Failure Modes

| Condition             | Behavior     |
| --------------------- | ------------ |
| Wappalyzer init fails | Fatal error  |
| No fingerprints       | Fatal error  |
| Empty body + headers  | Emit nothing |
| HTTP timeout          | Skip URL     |

---
