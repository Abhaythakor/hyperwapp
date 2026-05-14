# Baseline Performance Report: HyperWapp

## Test Environment
- **OS:** Linux
- **Arch:** amd64
- **CPU:** Intel(R) Celeron(R) 3205U @ 1.50GHz (2 cores)
- **Go Version:** 1.25.5

## Benchmark Results

### Detection Engine (`detect`)
| Metric | Value |
|--------|-------|
| Throughput | ~8.07 ms/op |
| Allocations | 20783 B/op |
| Allocs/op | 37 |

*Note: The high ns/op is primarily due to the large number of regex patterns evaluated by wappalyzergo.*

### FFF Parser (`input/fff`)
| Metric | Value |
|--------|-------|
| Throughput | ~0.64 ms/op |
| Allocations | 15982 B/op |
| Allocs/op | 96 |

## Resource Usage Observations

### CPU Profiling
- **Regex Matching:** ~23.6% of time spent in `matchString`.
- **GC Overhead:** ~18.6% in `runtime.scanobject`, indicating significant memory pressure.
- **Regex Compilation:** Significant time spent in `regexp/syntax.(*compiler)` even though initialization is outside the hot loop (likely due to internal lazy compilation or just high volume of calls).

### Memory Profiling
- **Regex Syntax:** >80% of total allocated space is related to regex compilation and simplification.
- **Wappalyzer Initialization:** `loadFingerprints` is the primary source of initial allocations.

## Conclusion
The application is currently **CPU-bound** by the regex engine and **Runtime-bound** by GC pressure from frequent small allocations. Online scanning is further handicapped by the lack of HTTP connection reuse.
