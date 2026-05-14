# Concurrency Map: HyperWapp

## Goroutine Structure

### 1. Main Goroutine
- Manages CLI lifecycle.
- Initializes engines and trackers.
- Spawns producer and consumer goroutines.
- Waits for completion or interrupt signal.

### 2. Result Handler Goroutine (`handleResults`)
- **Quantity:** 1
- **Role:** Consumes results from `resultCh`, performs Nuclei tag mapping, and writes to CLI and/or files.
- **Synchronization:** Synchronous writes to CLI and file (protected by mutex in some writers like `JSONLWriter`).

### 3. Worker Pool Goroutines
- **Quantity:** `concurrency` (default: 2 * CPU cores)
- **Role:** Performs the heavy lifting — fetching URLs (online) or running the Wappalyzer regex engine (offline/online/proxy).
- **Communication:** Reads from input channels, writes to `resultCh`.

### 4. Producer Goroutines
- **Quantity:** 1 (usually)
- **Role:** Parses input files or directories and feeds the worker pool.
- **Offline Mode:** `ParseOffline` starts a goroutine that walks the directory and sends `*model.OfflineInput` to the channel.

### 5. Progress Refresh Goroutine
- **Quantity:** 1
- **Role:** Updates the terminal UI at a fixed interval (4Hz).

## Channels and Buffering

| Channel | Type | Buffer Size | Source | Destination |
|---------|------|-------------|--------|-------------|
| `offlineInputCh` | `*model.OfflineInput` | 1000 | `ParseOffline` | `runOffline` |
| `offlineWorkerInputCh` | `*model.OfflineInput` | 2000 | `runOffline` | Workers |
| `resultChWorker` | `[]model.Detection` | 5000 | Workers | `handleResults` |
| `targetCh` (Online) | `model.Target` | 0 (unbuffered) | `runOnline` | Workers |
| `resultChWorker` (Online) | `[]model.Detection` | 0 (unbuffered) | Workers | `handleResults` |

## Synchronization Primitives
- **`sync.WaitGroup`:** Used to wait for worker pools to finish.
- **`sync.Pool`:** Used for `model.OfflineInput` to reduce allocation overhead.
- **`sync.Mutex`:** Used in `output` writers to ensure thread-safe writes to shared files.
- **`atomic.Uint32`:** Used in `progress.Tracker` for thread-safe count updates.
- **`context.Context`:** Used for graceful shutdown on `os.Interrupt`.

## Observed Bottlenecks / Risks
1. **Unbuffered Online Channels:** `targetCh` and `resultChWorker` in `runOnline` are unbuffered, which might lead to unnecessary worker blocking.
2. **Single-Threaded Result Handling:** All detections pass through one goroutine for Nuclei mapping and output. While this avoids complex locking in writers, it could be a bottleneck if output I/O is slow (though `bufio` helps).
3. **HTTP Client Creation:** `FetchOnline` creates a new `http.Client` for every request, preventing connection reuse.
