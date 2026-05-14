# Improvement Summary: HyperWapp Optimization

## Performance Gains

### 1. Detection Engine (`detect`)
- **Execution Time:** Reduced from ~8.07 ms/op to **~6.87 ms/op** (**15% improvement**).
- **Memory Allocation:** Reduced from 20,783 B/op to **16,454 B/op** (**20% reduction**).
- **Impact:** Higher throughput for tech detection across all modes.

### 2. FFF Parser (`input/fff`)
- **Execution Time:** Reduced from ~0.64 ms/op to **~0.29 ms/op** (**54% improvement**).
- **Impact:** Significant speedup in the discovery and ingestion phase for large FFF response directories.

## Key Changes Implemented

### Concurrency & Scalability
- **Connection Pooling:** Switched from per-request `http.Client` to a shared, tuned singleton client in `online` mode. This enables TCP connection reuse and eliminates TLS handshake overhead for every target.
- **Buffered Pipelines:** Added depth to critical channels (1000-2000 items) to prevent worker starvation and decouple production from consumption.
- **Parallel Tag Mapping:** Offloaded Nuclei tag mapping from the single-threaded result handler to the parallel worker pool. This prevents output I/O from being blocked by CPU-bound regex mapping.
- **Semaphore Control:** Implemented semaphore-based concurrency in parsers to prevent goroutine explosion when dealing with thousands of directories.

### Reliability & Observability
- **Context Propagation:** Updated the entire execution chain (Parsers -> Fetchers -> Workers) to accept and respect `context.Context`. HyperWapp now shuts down immediately and gracefully on Ctrl+C.
- **Graceful Shutdown:** All loops now check `ctx.Done()`, ensuring buffers are flushed and resources released correctly during interruption.

### Memory Efficiency
- **Object Pooling:** Added `sync.Pool` for large (1MB) buffers used in line counting during the discovery phase.
- **Pre-allocation:** Optimized `detect.Engine` to pre-allocate result slices based on expected fingerprint matches.
- **Map Reuse:** Refactored offline parsers to reuse the `Headers` map from the `OfflineInput` pool instead of allocating a new map for every target, significantly reducing GC pressure.

## Conclusion
HyperWapp is now a more robust, faster, and memory-efficient tool. The architectural changes provide a solid foundation for further scaling and maintainability.
