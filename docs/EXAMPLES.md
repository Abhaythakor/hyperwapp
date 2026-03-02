# HyperWapp Examples 💡

This page provides multiple real-world scenarios for using HyperWapp effectively.

---

## 1. Online Scanning Scenarios

### Basic Single URL
Detect technologies on a single site with default settings.
```bash
hyperwapp -u https://news.airbnb.com
```

### High-Speed Bulk Scan
Scan a list of 1 million URLs using 200 concurrent workers.
```bash
hyperwapp -l urls.txt -c 200 -f jsonl -o my_scan.jsonl
```

### Piping from Subfinder/Httpx
Combine HyperWapp with other tools in your pipeline.
```bash
subfinder -d airbnb.com | httpx | hyperwapp -c 50
```

### Proxy Mode (Passive Scanning)
Start a local proxy server and configure your browser to use it. HyperWapp will scan all traffic in real-time.
```bash
hyperwapp --proxy :8080
```

---

## 2. Offline Scanning Scenarios

### Katana Recursive Discovery
Scan a directory containing deeply nested Katana response files.
```bash
hyperwapp -offline ./katana-output/ -c 100 -f csv -o katana_techs.csv
```

### FFF Standard Output
Scan a directory structured by the FFF tool (`domain/hash.headers` and `domain/hash.body`).
```bash
hyperwapp -offline ./fff-data/ -c 50 -f json -o fff_summary.json
```

### Multi-Domain Aggregation
Group all detections by domain even if you have thousands of individual asset files.
```bash
hyperwapp -offline ./responses/ --domain -f md -o report.md
```

---

## 3. Advanced Operational Scenarios

### Handling Interrupted 10M Target Scans
If your server crashes or you lose power during a massive scan, use the `--resume` flag to pick up exactly where you left off.
```bash
# First run
hyperwapp -offline ./massive_dump/ -c 150 -f jsonl -o data.jsonl --resume

# ... system crashes ...

# Restart run (it will skip all completed items and show original total)
hyperwapp -offline ./massive_dump/ -c 150 -f jsonl -o data.jsonl --resume
```

### CPU-Limited Scan
Run a background scan while keeping your server responsive for other tasks by limiting HyperWapp to 4 CPU cores.
```bash
hyperwapp urls.txt -c 50 --cpus 4
```

### Live JSONL Monitoring
Standard JSON is hard to read while it's being written. Use JSONL and `tail` to see data in real-time.
```bash
# Terminal 1
hyperwapp urls.txt -f jsonl -o live.jsonl

# Terminal 2
tail -f live.jsonl | jq .technology
```

---

## 4. Custom Input Configuration Scenarios

### Parsing Custom JSON (GJSON Paths)
If you have a JSON file from a custom tool with a different structure, use GJSON paths to map fields.
```bash
# Example JSON line: {"target_url": "...", "resp": {"header_map": {...}, "html": "..."}}
# Config (config.yaml):
# format: "json"
# json:
#   url_path: "target_url"
#   headers_path: "resp.header_map"
#   body_path: "resp.html"

hyperwapp -offline data.jsonl --input-config config.yaml
```

### Parsing Custom Log Files (Regex Blocks)
If your data is stored in a text file separated by a specific string, use the Regex parser.
```bash
# Example Log:
# URL: https://example.com
# [HEADERS]
# Server: Apache
# [BODY]
# <html>...</html>
# ---
# URL: https://target.com
# ...

# Config (config.yaml):
# format: "regex"
# record_separator: "\n---\n"
# regex:
#   url_regex: "URL: (https?://[^\\s\n]+)"
#   headers_regex: "(?s)\\[HEADERS\\]\\n(.*?)\n\\[BODY\\]"
#   body_regex: "(?s)\\[BODY\\]\\n(.*)"

hyperwapp -offline my_logs.txt --input-config config.yaml
```
