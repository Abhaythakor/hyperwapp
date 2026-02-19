# PART 4

## Offline Parsing, Execution Pipeline, Directory Structure, Error Handling

---

## 28. OFFLINE MODE (DETAILED, WAPPALYZER-READY)

### Purpose of offline mode

Offline mode exists to **reconstruct enough HTTP context** to allow Wappalyzer to work without network access.

Offline mode does **not** do detection itself.

---

## 29. Supported Offline Inputs

Offline input may come from:

- `fff`
- `katana`
- Raw HTTP dumps
- Saved HTML files

Supported content types:

- Headers only
- Body only
- Headers + body combined

Only **one file or stdin** is allowed.

---

## 30. Offline Flags Behavior

| Flag            | Effect               |
| --------------- | -------------------- |
| `-headers-only` | Only parse headers   |
| `-body-only`    | Only parse body      |
| `-auto`         | Parse both (default) |

Rules:

- Flags affect what is passed to Wappalyzer
- Parsing still attempts to detect structure
- Empty headers or body are allowed

---

## 31. Offline Parsing Strategy

### Core goal

Produce:

```go
map[string][]string   // headers
[]byte                // body
```

### Parsing state machine

```
START
 ↓
HEADER MODE
 ↓ (blank line)
BODY MODE
```

---

### Header detection rules

A line is considered a header if:

- It matches `Key: Value`
- Or starts with `HTTP/1.`

Headers are stored in canonical Go format:

```go
headers["Server"] = []string{"nginx"}
```

---

### Body detection rules

Everything after headers is treated as body:

- HTML
- JS
- CSS
- JSON

Stored as raw bytes.

---

## 32. Offline URL & Domain Handling

Offline inputs may not contain URLs.

Rules:

- If URL is known → populate `url`
- If only domain is known → populate `domain`
- If neither exists → domain = `unknown`

CSV and JSON must still be valid.

---

## 33. Offline → Wappalyzer Flow

```
Offline file
 ↓
Parse headers
 ↓
Parse body
 ↓
Apply flags
 ↓
Wappalyzer Fingerprint(headers, body)
 ↓
Detection records
```

Same mapping rules as online mode.

---

## 34. EXECUTION PIPELINE (FULL)

### High-level execution flow

```
CLI
 ↓
Config validation
 ↓
Input resolver
 ↓
Target normalization
 ↓
Worker pool
   ├─ Online fetch OR Offline parse
   ├─ Wappalyzer fingerprint
   └─ Emit detection records
 ↓
Aggregation (optional)
 ↓
Writers
 ↓
Exit
```

---

## 35. CONCURRENCY MODEL

- Concurrency at **target level**
- One target = one Wappalyzer execution
- Detection engine is stateless
- Writers are thread-safe

Default:

```
-threads = 10
```

---

## 36. PROGRESS TRACKING

Enabled automatically when:

- More than one target
- Not in quiet mode

Example:

```
[+] Total: 200
[+] Completed: 83
[+] Remaining: 117
```

Thread-safe counter updates.

---

## 37. FINAL DIRECTORY STRUCTURE (UPDATED)

```
hyperwapp/
├── main.go
├── go.mod
├── go.sum
│
├── cmd/
│   └── root.go
│
├── config/
│   ├── config.go
│   └── defaults.go
│
├── input/
│   ├── resolver.go
│   ├── normalize.go
│   ├── online.go
│   └── offline.go
│
├── detect/
│   ├── engine.go
│   └── wappalyzer.go
│
├── model/
│   ├── detection.go
│   ├── target.go
│   └── metadata.go
│
├── aggregate/
│   ├── domain.go
│   └── url.go
│
├── output/
│   ├── writer.go
│   ├── csv.go
│   ├── json.go
│   ├── txt.go
│   ├── md.go
│   └── cli.go
│
├── progress/
│   └── tracker.go
│
├── util/
│   ├── http.go
│   ├── time.go
│   └── logger.go
│
├── examples/
│   ├── urls.txt
│   └── offline.txt
│
└── README.md
```

---

## 38. ERROR HANDLING STRATEGY

| Condition            | Behavior              |
| -------------------- | --------------------- |
| Invalid URL          | Skip, warn            |
| HTTP timeout         | Skip, warn            |
| Offline parse error  | Warn with line number |
| Wappalyzer init fail | Fatal                 |
| Missing fingerprints | Fatal                 |
| Output write error   | Exit non-zero         |

Exit codes:

- `0` success
- `1` partial failure
- `2` fatal error

---

## 39. GUARANTEES THIS PLAN PROVIDES

- Uses **industry-grade detection**
- Zero fingerprint maintenance burden
- Stable CSV & JSON forever
- Offline and online parity
- Scales to large recon runs
- Easy future extensions

---

## 40. FUTURE EXTENSIONS (SAFE)

This design cleanly supports:

- Hybrid detection (Wappalyzer + custom rules)
- Version extraction
- IP / ASN enrichment
- Historical diffing
- Evidence hashing
- Streaming JSON

Without breaking schema.
