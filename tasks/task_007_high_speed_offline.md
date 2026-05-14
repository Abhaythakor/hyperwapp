# Task 007: Rewrite Offline Pipeline for Extreme Speed

## Problem Description
Current offline JSON scanning is limited by single-threaded IO and JSON parsing, combined with heavy regex matching on large bodies.

## Root Cause
- Sequential reading of 17GB file.
- Single-threaded line parsing.
- Blindly running 3,000+ regexes on every body.

## Proposed Solution (The Rewrite)
1. **Parallel IO:** Read the file in 64MB blocks using multiple goroutines (if on SSD) or one high-speed sequential reader that dispatches chunks.
2. **Parallel JSON Extraction:** Use a pool of workers to find line boundaries in blocks and extract fields using a fast, custom byte-search rather than full GJSON decoding.
3. **Pre-Filter Scan:** Run a single `bytes.Contains` pass for common technology "hints" before invoking `wappalyzergo`.
4. **Strict Concurrency Control:** Fix the worker pool to exactly `runtime.NumCPU()` to prevent context-switching penalties.

## Expected Performance Impact
Extreme (Goal: 10x - 20x improvement, matching Online speed).

## Status
Pending (Researching Implementation Details)
