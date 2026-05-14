# Research: Concurrency Patterns in Go

## 1. Worker Pools
- **Pattern:** Fixed-size pool of goroutines consuming from a shared channel.
- **Benefits:** Prevents goroutine explosion, controls resource usage (CPU/RAM).
- **Current State:** HyperWapp uses worker pools but some channels are unbuffered.
- **Best Practice:** Use buffered channels to allow producers to stay ahead of consumers without blocking, but keep buffer sizes reasonable to avoid excessive memory usage.

## 2. Pipelines
- **Pattern:** Multiple stages of processing where each stage runs in its own pool of goroutines and communicates via channels.
- **Example:** `Input Parser -> Worker (Fetcher/Detector) -> Result Handler`.
- **Optimization:** Each stage should be independently scalable. If `Result Handler` is a bottleneck, it can be parallelized if order doesn't matter.

## 3. Context Usage
- **Requirement:** Pass `context.Context` through all stages for cancellation and timeout management.
- **Current State:** HyperWapp uses `signal.NotifyContext` in `rootCmd.Run` but doesn't propagate it fully to all sub-components (like `FetchOnline`).

## 4. Error Handling in Concurrent Systems
- **Strategy:** Use an error channel or `errgroup` to collect and handle errors from multiple goroutines gracefully.
- **Optimization:** For a reconnaissance tool, often we want to log and continue rather than stop everything on a single target failure.
