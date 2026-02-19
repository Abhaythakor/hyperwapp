# HyperWapp 🚀

[![Go Report Card](https://goreportcard.com/badge/github.com/Abhaythakor/hyperwapp)](https://goreportcard.com/report/github.com/Abhaythakor/hyperwapp)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**HyperWapp** is a high-performance, massively scalable CLI reconnaissance utility designed to detect web technologies using Wappalyzer fingerprints. Whether you are scanning a single URL or an offline dump of **10,000,000 targets**, HyperWapp handles it with ease using disk-backed streaming and zero-RAM aggregation.

Built upon the powerful foundation of [ProjectDiscovery's WappalyzerGo](https://github.com/projectdiscovery/wappalyzergo).

---

## ✨ Key Features

*   🔍 **Powered by Wappalyzer**: Uses industry-standard fingerprints for high accuracy.
*   📦 **Massive Scalability**: Optimized for **10M+ targets** using disk-backed JSONL streaming.
*   📁 **Advanced Offline Mode**: Recursively parse directory structures from **Katana**, **FFF**, or raw HTTP dumps.
*   ⏯️ **Checkpoint & Resume**: Instantly restart interrupted scans without losing data or re-discovering targets.
*   ⚡ **Full Performance Control**: Independent control over **Concurrency** (Goroutines) and **Parallelism** (CPU Cores).
*   📊 **Real-time Status Footer**: Live RPS, success/error counts, and percentage tracking in your terminal.
*   💾 **Multiple Formats**: Export to CSV, JSON, TXT, Markdown, or real-time JSONL.

---

## 🛠 Installation

### From Source
```bash
go install github.com/Abhaythakor/hyperwapp@latest
```

---

## ⚡ Quick Start

### 1. Online Scan (Single URL)
```bash
hyperwapp https://example.com
```

### 2. URL List with High Concurrency
```bash
cat urls.txt | hyperwapp -c 100 -f jsonl -o results.jsonl
```

### 3. Advanced Offline Scan (Recursive)
```bash
hyperwapp -offline ./katana_responses/ -c 50 -f csv -o summary.csv
```

### 4. Update Fingerprints
```bash
hyperwapp --update
```

---

## 📖 Documentation

For detailed guides, please see:
*   [Detailed Usage & Flags](./docs/USAGE.md)
*   [Real-world Examples](./docs/EXAMPLES.md)
*   [Massive Scale Tuning (10M+ Targets)](./docs/PERFORMANCE.md)
*   [Understanding Offline Formats](./docs/OFFLINE_FORMATS.md)

---

## 🙏 Credits

This tool would not be possible without the incredible work of:
- [ProjectDiscovery](https://projectdiscovery.io/) for their [wappalyzergo](https://github.com/projectdiscovery/wappalyzergo) library.

---

## 📜 License
HyperWapp is released under the [MIT License](LICENSE).
