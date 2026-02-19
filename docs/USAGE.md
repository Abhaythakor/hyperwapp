# HyperWapp Usage & Flags 📖

This document provides a detailed breakdown of every command-line flag available in HyperWapp.

---

## 1. Input Flags

### `-offline`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Enables Offline Mode. Instead of treating the input as a URL or URL list, HyperWapp will recursively walk the provided directory path to find and parse stored HTTP responses (Katana, FFF, etc.).

### `-auto`
*   **Type:** Boolean
*   **Default:** `true`
*   **Description:** Automatically detects technologies using both HTTP headers and the response body.

### `-headers-only`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Forces the engine to only look at HTTP headers. This is faster but less accurate for client-side frameworks like React or Vue.

### `-body-only`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Forces the engine to only look at the response body.

---

## 2. Output Style Flags

### `--all`
*   **Type:** Boolean
*   **Default:** `true`
*   **Description:** Outputs one record per URL. This is the standard behavior for reconnaissance.

### `--domain`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Aggregates all detections by their root domain. Useful for high-level summaries. 
*   *Note:* In this mode, the tool will wait until the scan is finished before producing the final aggregated output.

---

## 3. Export & Format Flags

### `-o, --output <file>`
*   **Type:** String
*   **Description:** Path to the file where results should be saved. If omitted, results are only printed to the CLI.

### `-f, --format <format>`
*   **Type:** String
*   **Default:** `cli`
*   **Options:** `csv`, `json`, `jsonl`, `txt`, `md`, `cli`
*   **Descriptions:**
    *   `csv`: Standard spreadsheet-ready format.
    *   `json`: A single valid JSON array (not recommended for 1M+ targets).
    *   `jsonl`: **(Recommended)** JSON Lines. Each detection is its own line. Best for big data.
    *   `txt`: Human-readable plain text.
    *   `md`: Formatted Markdown report.

---

## 4. Performance Flags

### `-c, --concurrency <int>`
*   **Type:** Integer
*   **Default:** `10`
*   **Description:** Number of concurrent workers (Goroutines) to run. 
*   *Note:* `-t` and `--threads` are supported as aliases.

### `--cpus <int>`
*   **Type:** Integer
*   **Default:** `0` (All available)
*   **Description:** Limits the number of physical CPU cores the Go runtime will use.

### `--timeout <int>`
*   **Type:** Integer
*   **Default:** `10`
*   **Description:** HTTP timeout in seconds for online scanning.

---

## 5. UI & Logging Flags

### `--no-color` / `--mono`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Disables ANSI colors in the CLI output.

### `-v, --verbose`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Enables debug logging. Use this if you are having issues with offline format detection.

### `--silent`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Display results only. Suppresses the progress tracker and all informational logs. Useful for piping output to other tools.

---

## 6. Advanced Flags

### `--resume`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Enables the checkpoint system. It will load progress from `.hyperwapp.resume` and skip already processed items.

### `--update`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Downloads the latest technology fingerprints directly from the ProjectDiscovery WappalyzerGo repository.

### `--version`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Displays the current version of HyperWapp and the timestamp of the last fingerprint update.
