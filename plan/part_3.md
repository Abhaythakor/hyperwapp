# PART 3

## CSV & JSON Output (Final) and Domain Aggregation

---

## 21. Output Philosophy (Critical)

- **CSV and JSON are authoritative**
- CLI, TXT, and MD are presentation only
- No information loss is allowed in CSV or JSON
- Aggregation never modifies raw detections

---

## 22. CSV EXPORT (FINAL, STABLE)

### CSV Purpose

- Machine-ingestible
- SIEM-ready
- Spreadsheet-friendly
- One row = one detection fact

---

### CSV Header (Fixed Order)

```
domain,url,technology,source,path,evidence,confidence,timestamp
```

This header **never changes**.
New fields are appended in future versions only.

---

### CSV Field Definitions

| Field      | Meaning                                            |
| ---------- | -------------------------------------------------- |
| domain     | Root domain                                        |
| url        | Full URL (empty in domain mode or unknown offline) |
| technology | Detected technology                                |
| source     | wappalyzer / wappalyzer-header / wappalyzer-body   |
| path       | fingerprint                                        |
| evidence   | wappalyzergo                                       |
| confidence | high / medium / low                                |
| timestamp  | RFC3339                                            |

---

### CSV Example (All Mode)

```
example.com,https://example.com,Cloudflare,wappalyzer,fingerprint,wappalyzergo,high,2026-01-11T15:22:04Z
example.com,https://example.com,React,wappalyzer,fingerprint,wappalyzergo,high,2026-01-11T15:22:04Z
```

---

### CSV Behavior in Domain Mode

Rules:

- `url` column is empty
- One row per detection
- No deduplication

```
example.com,,Cloudflare,wappalyzer,fingerprint,wappalyzergo,high,2026-01-11T15:22:04Z
```

---

## 23. JSON EXPORT (FINAL, STRUCTURED)

### JSON Goals

- Structured but lossless
- Friendly to jq, Elastic, BigQuery
- Explicit metadata

---

### JSON Top-Level Structure

```json
{
  "meta": {
    "tool": "hyperwapp",
    "version": "1.0.0",
    "generated_at": "2026-01-11T15:22:04Z",
    "mode": "all",
    "input_type": "online"
  },
  "results": []
}
```

---

### JSON Result Object (All Mode)

```json
{
  "domain": "example.com",
  "url": "https://example.com",
  "detections": [
    {
      "technology": "Cloudflare",
      "source": "wappalyzer",
      "path": "fingerprint",
      "evidence": "wappalyzergo",
      "confidence": "high",
      "timestamp": "2026-01-11T15:22:04Z"
    }
  ]
}
```

---

### JSON Result Object (Domain Mode)

```json
{
  "domain": "example.com",
  "urls": ["https://example.com", "https://blog.example.com"],
  "detections": [
    {
      "technology": "React",
      "source": "wappalyzer",
      "path": "fingerprint",
      "evidence": "wappalyzergo",
      "confidence": "high",
      "timestamp": "2026-01-11T15:22:04Z"
    }
  ]
}
```

---

## 24. JSON Rules (Strict)

- No deduplication inside `detections`
- Same technology may appear multiple times
- Order is preserved
- Timestamps are mandatory

---

## 25. DOMAIN AGGREGATION LOGIC

### What aggregation does

- Group detections by domain
- Collect unique URLs
- Present domain-level view

---

### What aggregation does NOT do

- Remove detections
- Merge technologies
- Modify timestamps
- Change confidence

---

### Aggregation Flow

1. Group Detection records by `domain`
2. For each domain:
   - Build URL set
   - Attach all detections

3. Output aggregated structure

---

## 26. EXPORT PIPELINE (UPDATED)

### Internal flow

```
Detection Engine
   ↓
Detection Records
   ↓
Aggregation (optional)
   ↓
Writer (CSV / JSON / TXT / MD / CLI)
```

Detection engine is unaware of output format.

---

## 27. Streaming vs Buffered Output

### CSV

- Supports streaming
- Writes row-by-row
- Safe for large scans

### JSON

- Buffered
- Written at end
- Always valid JSON
