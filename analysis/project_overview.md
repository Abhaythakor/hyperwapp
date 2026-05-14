# Project Overview: HyperWapp

## Purpose
HyperWapp is a high-performance technology reconnaissance utility. It leverages Wappalyzer fingerprints to identify web technologies from various input sources.

## Core Components
- **CLI (`cmd/`):** Built with Cobra, manages flags, initialization, and the main execution loop.
- **Detection Engine (`detect/`):** Wraps `wappalyzergo` to provide technology identification.
- **Input System (`input/`):** Handles diverse input formats (Online URLs, Offline directories/files like FFF, Katana, Raw HTTP).
- **Output System (`output/`):** Provides multiple export formats (CLI, JSON, JSONL, CSV, TXT, MD).
- **Model (`model/`):** Defines core data structures like `OfflineInput` and `Detection`.
- **Progress Tracking (`progress/`):** Provides a real-time, throttled UI for scan progress.

## Execution Modes
1. **Online Scan:** Actively fetches content from a list of URLs.
2. **Offline Scan:** Recursively parses local files containing HTTP responses.
3. **Proxy Mode:** Passively detects technologies by intercepting traffic.

## Key Dependencies
- `github.com/projectdiscovery/wappalyzergo`: Core detection library.
- `github.com/spf13/cobra`: CLI framework.
- `github.com/tidwall/gjson`: Fast JSON parsing.
- `gopkg.in/yaml.v3`: Configuration parsing.

## Performance Features
- **Object Pooling:** `sync.Pool` is used for `OfflineInput` to reduce GC pressure.
- **Buffered I/O:** `bufio.Writer` with large buffers (4MB) for file output.
- **Concurrent Workers:** Goroutine-based worker pools for both parsing and detection.
- **Throttled UI:** Progress updates are capped at 4Hz to save CPU.
