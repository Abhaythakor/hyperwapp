# PART 10

## Performance, Memory Model & Scaling Strategy

---

## 10.1 Performance Goals

The tool must:

- Handle **hundreds of thousands** of offline files
- Avoid loading large datasets into memory
- Scale linearly with input size
- Never block on a single slow file
- Remain usable on low-memory systems

Performance is a **first-class requirement**, not an optimization.

---

## 10.2 Core Performance Principles

1. **Streaming over buffering**
2. **Bounded concurrency**
3. **Zero global aggregation by default**
4. **Work per target, not per byte**
5. **Fail fast, continue always**

---

## 10.3 Memory Ownership Rules

These rules prevent accidental memory blowups.

### Allowed in memory

- One `OfflineInput` at a time per worker
- One HTTP response per worker
- Small metadata maps

### Forbidden in memory

- Entire fff directory trees
- Entire katana datasets
- Full CSV or JSON output buffers (except final JSON)

---

## 10.4 Execution Model (High-Level)

```
Input Enumerator
   ↓
Work Queue (bounded)
   ↓
Worker Pool (N workers)
   ↓
Detection (Wappalyzer)
   ↓
Output Writers (streaming)
```

No stage blocks indefinitely.

---

## 10.5 Input Enumeration Strategy

### Online mode

- Enumerate URLs line-by-line
- Push each URL to queue
- No preloading

### Offline mode

- Enumerate **file paths**, not contents
- Emit work items incrementally
- Never walk entire trees into memory

---

## 10.6 fff Enumeration Strategy (Critical)

### Correct approach

```
Walk directory
 ├─ discover headers/body pairs
 ├─ emit work item immediately
 └─ discard path references
```

### Incorrect approach (forbidden)

- Building a full hash → file map for entire domain
- Storing directory trees in memory

---

## 10.7 fff Streaming Pseudocode

```go
filepath.WalkDir(domainPath, func(p string, d fs.DirEntry, err error) error {
	if isHeadersFile(p) {
		body := findMatchingBody(p)
		workQueue <- FFFWorkItem{
			HeadersPath: p,
			BodyPath: body,
		}
	}
	return nil
})
```

Each work item is processed independently.

---

## 10.8 Katana Enumeration Strategy

Katana files are **already discrete**.

Strategy:

- Emit one work item per `.txt` file
- Parse and discard immediately
- Never keep katana file contents after processing

---

## 10.9 Worker Pool Design

### Worker responsibilities

Each worker:

1. Receives one work item
2. Reads headers and/or body
3. Builds `OfflineInput`
4. Runs Wappalyzer
5. Emits Detection records
6. Releases memory

---

### Worker pool constraints

- Fixed size
- Default: `threads = 10`
- Upper bound enforced (e.g. 50)

This avoids file descriptor exhaustion.

---

## 10.10 Wappalyzer Memory Behavior

WappalyzerGo:

- Loads fingerprints once
- Reuses internal structures
- Stateless per call

Implications:

- One shared instance per process
- Safe to call concurrently
- No need to reinitialize per worker

---

## 10.11 Output Writing Strategy

### CSV (streaming)

- Write one row per detection
- Flush periodically
- No buffering

### JSON (buffered but bounded)

- Aggregate per domain (domain mode)
- Aggregate per URL (all mode)
- Write final JSON at end

For extremely large datasets, document that CSV is preferred.

---

## 10.12 Backpressure Handling

### Problem

Slow disk or slow stdout can stall workers.

### Solution

- Buffered output channel
- Bounded size
- If full:
  - Workers block briefly
  - No data loss
  - Progress continues

---

## 10.13 Progress Tracking at Scale

Progress counters must:

- Be atomic
- Avoid locks in hot paths
- Update on **work item completion**, not file discovery

Example:

```
[+] Total: 120,345
[+] Completed: 78,002
[+] Remaining: 42,343
```

---

## 10.14 Garbage Collection Strategy

To reduce GC pressure:

- Reuse byte buffers where possible
- Avoid converting body to string
- Avoid large temporary slices
- Let large byte slices go out of scope quickly

---

## 10.15 Failure Isolation

A failure must never stop the pipeline.

Rules:

- One bad file ≠ fatal
- One malformed response ≠ fatal
- One slow file ≠ stall

Each work item is isolated.

---

## 10.16 Performance Testing Strategy

### Benchmarks to include

- fff dataset with:
  - 10k files
  - 100k files

- katana directory with:
  - 10k responses

- body-only JS assets

Measure:

- Time to completion
- Peak memory usage
- Throughput (items/sec)

---

## 10.17 Expected Performance Characteristics

On a modern laptop:

- fff (100k files): minutes, not hours
- Memory usage: stable, < 500MB
- CPU bound, not I/O bound
- Linear scaling with thread count

---

## 10.18 When to Recommend Flags

Document recommendations:

- Large offline dataset:

  ```
  -format csv -no-color
  ```

- Memory constrained system:

  ```
  -threads 5
  ```

---

## 10.19 What PART 10 Guarantees

After implementing this part:

- Tool scales to real recon workloads
- No surprise memory spikes
- Predictable runtime
- Safe parallelism
- Clear operational guidance
