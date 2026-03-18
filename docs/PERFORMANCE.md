# Tuning for Massive Scale (10M+ Targets) 🚀

HyperWapp is designed to handle "Internet-scale" scans. However, when moving beyond 1 million targets, system configuration becomes critical.

## 1. System File Limits (`ulimit`)
Each concurrent worker (concurrency) may hold an open file or network socket. If you use `-c 1000`, you will likely hit the default Linux limit of 1024 open files.

**Recommendation:** Increase your limit before starting a large scan.
```bash
ulimit -n 100000
```

## 2. Choosing the Right Format
At 10,000,000 targets, your output file will be massive (multiple Gigabytes).

*   ❌ **Avoid `-f json`**: Writing a single JSON array requires a massive finalization step. If the tool is interrupted, the file is corrupted.
*   ✅ **Use `-f jsonl`**: This is the most efficient format. It writes every detection to disk immediately. It is crash-proof and uses zero RAM for writing.
*   ✅ **Use `-f csv`**: Also very efficient and streams directly to disk.

## 3. Concurrency vs. Parallelism
*   **Concurrency (`-c`)**: This is the number of Goroutines. Set this high (50-500) if you are waiting on Disk I/O or the Internet.
*   **Parallelism (`--cpus`)**: Wappalyzer is CPU-intensive (Regex). If you have a 32-core server, Go will use all 32 cores by default. If the CPU hits 100% and the system lags, use `--cpus` to limit HyperWapp to a subset of your cores.

## 4. Disk Speed (Offline)
For offline scans of 10 million files, your **SSD speed** is usually the bottleneck. 

HyperWapp is optimized for extreme throughput:
*   **4MB RAM Write Buffer:** All detections are buffered in 4MB blocks to reduce expensive OS write calls. This is ideal for 100GB+ output files.
*   **10,000-Item Pipeline Queue:** A massive look-ahead queue ensures that your workers are never "starved" for data from the disk reader.
*   **Zero-Copy Parallelism:** Each worker uses an independent engine instance to avoid global locks and saturate all CPU cores.

## 5. Memory Usage
HyperWapp uses **Disk-Backed Streaming**. This means even for 10 million targets, the RAM usage should remain stable (typically 500MB - 1GB depending on your queue size). 
*   **RAM Buffer Control:** The tool typically uses `10,000 * (average response size)` in RAM for the pipeline.
*   If you see RAM climbing, it is likely due to the `--domain` aggregation phase at the very end. 
*   For the lowest possible RAM footprint, use `-f jsonl` without the `--domain` flag.
