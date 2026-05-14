# Research: High-Speed Offline Scanning for Large JSONL

## 1. I/O Bottleneck: Single-Threaded Sequential Read
- **Problem:** Currently, one goroutine reads the 17GB file line-by-line using `bufio.Scanner` or `ReadBytes`. This is slow on HDDs.
- **Solution:** **Parallel Block Processing**. 
    - Divide the file into chunks (e.g., 64MB).
    - Use multiple workers to find line boundaries within blocks.
    - Process blocks in parallel to keep CPU cores saturated while HDD performs sequential reads.

## 2. Parsing Bottleneck: The 'JSON Tax'
- **Problem:** `gjson` is fast but decoding full HTML bodies from JSON strings involves massive string-to-byte conversions and escaping.
- **Solution:** **SIMD-accelerated JSON parsing** or **Pre-extraction**.
    - If the JSON format is fixed, we can search for byte offsets of `"body":"` and `"headers":{` directly to avoid full JSON decoding.

## 3. Execution Bottleneck: Blind Regex Execution
- **Problem:** Wappalyzer runs ~3,000 regexes against every body.
- **Solution:** **Aho-Corasick Pre-filtering**.
    - Build a map of "Required Strings" (e.g., tech "WordPress" requires string "wp-content").
    - Run one single-pass scan over the body for all keywords.
    - Only run the complex regexes for the technologies whose keywords were found.
    - *Challenge:* Requires forking or wrapping `wappalyzergo` to expose fingerprint keywords.

## 4. Hardware Alignment (2-Core CPU)
- **Constraint:** Context switching between 300 threads is killing performance.
- **Solution:** Use a strictly limited worker pool matching `runtime.NumCPU()`. 
- **Refinement:** Use a **Work-Stealing Scheduler** pattern for the pipeline.
