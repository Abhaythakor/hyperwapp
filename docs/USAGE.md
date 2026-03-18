# HyperWapp Usage & Flags 📖

This document provides a detailed breakdown of every command-line flag available in HyperWapp.

---

## 1. Input Flags

### `-u, --url <string>`
*   **Type:** String
*   **Description:** Single URL to scan. 
*   **Example:** `hyperwapp -u https://example.com`

### `-l, --list <file>`
*   **Type:** String
*   **Description:** Path to a file containing a list of URLs to scan (one per line).
*   **Example:** `hyperwapp -l urls.txt`

### `-offline`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Enables Offline Mode. Instead of treating the input as a URL or URL list, HyperWapp will recursively walk the provided directory path to find and parse stored HTTP responses (Katana, FFF, etc.).
*   **Example:** `hyperwapp -offline ./data/`

### `--proxy <address>`
*   **Type:** String
*   **Description:** Starts a proxy server on the specified address (e.g., `:8080`) to passively scan traffic from a browser or other tools.
*   **SSL/TLS Instructions:** 
    1.  When you start the proxy, HyperWapp will create a file called `hyperwapp-ca.crt` in your current folder.
    2.  To avoid "Connection is not private" errors, you **must** import this file into your browser's **Trusted Root Certificate Authorities**.
    3.  **Chrome/Edge:** `Settings -> Security -> Manage Certificates -> Authorities -> Import`.
    4.  **Firefox:** `Settings -> Privacy & Security -> Certificates -> View Certificates -> Authorities -> Import`.
*   **Example:** `hyperwapp --proxy :8080`

### `--input-config <file>`
*   **Type:** String
*   **Description:** Path to a YAML configuration file for custom input parsing. Supports GJSON paths for JSON files and Regex patterns for any text-based logs or reports.
*   **Example:** `hyperwapp -offline ./custom_logs/ --input-config config.yaml`

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
*   **Description:** Path to the file where results should be saved. Even when saving to a file, results are still printed to the CLI.

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
*   **Description:** Limits the number of physical CPU cores the Go runtime will use (GOMAXPROCS).

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

## 6. Utility Flags

### `--resume`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Enables the checkpoint system. It will load progress from `.hyperwapp.resume` and skip already processed items.

### `--update`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Updates both the Wappalyzer fingerprints (from ProjectDiscovery) and the HyperWapp binary itself (using `go install`).

### `--version`
*   **Type:** Boolean
*   **Default:** `false`
*   **Description:** Displays the current version of HyperWapp and the timestamp of the last fingerprint update.
