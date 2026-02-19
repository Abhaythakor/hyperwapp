# Installation Guide 🛠️

HyperWapp is written in Go and designed to be portable across Linux, macOS, and Windows.

---

## 1. Prerequisites

You must have **Go 1.21** or higher installed on your system.
```bash
go version
```

---

## 2. Installing from GitHub

The recommended way to install HyperWapp is using `go install`:

```bash
go install github.com/Abhaythakor/hyperwapp@latest
```

Ensure your Go bin directory is in your system's `PATH`:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

---

## 3. Building from Source

If you want to modify the code or build it manually:

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/Abhaythakor/hyperwapp.git
    cd hyperwapp
    ```

2.  **Download dependencies:**
    ```bash
    go mod download
    ```

3.  **Build the binary:**
    ```bash
    go build -o hyperwapp main.go
    ```

4.  **Move to your bin folder:**
    ```bash
    mv hyperwapp /usr/local/bin/
    ```

---

## 4. Updating Fingerprints

HyperWapp uses the `wappalyzergo` engine. While the tool is self-contained, fingerprints are periodically updated. You can update them directly within HyperWapp:

```bash
hyperwapp --update
```

This will download the latest `fingerprints.json` to your local config directory.

---

## 5. System Requirements

*   **RAM:** 512MB minimum (optimized for low RAM usage).
*   **Disk:** SSD highly recommended for massive offline scans.
*   **OS:** Linux (preferred), macOS, Windows.
