# Performance Hotspots: HyperWapp

## 1. Detection Engine (Regex Matching)
- **Symptom:** High CPU usage in `wappalyzergo.(*CompiledFingerprints).matchString` and `regexp.(*Regexp).FindStringSubmatch`.
- **Root Cause:** Wappalyzer has thousands of fingerprints, each with one or more regex patterns. Every detection run iterates through a large set of these patterns against headers and body.
- **Impact:** Limits throughput to ~120-150 detections per second per core (based on `BenchmarkDetect` ~8ms/op).

## 2. HTTP Client Creation (Online Mode)
- **Symptom:** High latency and socket exhaustion in online scans.
- **Root Cause:** `FetchOnline` instantiates a new `http.Client` for every request.
- **Impact:** Prevents TCP connection reuse (Keep-Alive), significantly increasing overhead for each request due to repeated TLS handshakes.

## 3. Memory Allocations (Regex & JSON)
- **Symptom:** High memory traffic in `regexp/syntax` and `encoding/json`.
- **Root Cause:** 
    - Initialization of `wappalyzergo` is very expensive in terms of memory allocations (compiling thousands of regexes).
    - Frequent map iterations and small object allocations during detection.
- **Impact:** Increased GC pressure, which shows up as `runtime.scanobject` in CPU profiles.

## 4. Unbuffered Channels (Online Mode)
- **Symptom:** Potential worker starvation or blocking.
- **Root Cause:** `targetCh` and `resultChWorker` in `runOnline` are unbuffered.
- **Impact:** Synchronizes producers and consumers unnecessarily, potentially slowing down the pipeline if any stage (fetching, detecting, or writing) hit a transient delay.

## 5. Single-Threaded Result Processing
- **Symptom:** Sequential bottleneck at the end of the pipeline.
- **Root Cause:** `handleResults` runs in a single goroutine and performs Nuclei mapping and multiple writes.
- **Impact:** If output I/O (especially to slow disks or remote filesystems) becomes slow, it can backpressure the entire worker pool.
