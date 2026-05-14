# Task 001: Shared HTTP Client & Connection Pooling

## Problem Description
The `FetchOnline` function creates a new `http.Client` for every request.

## Root Cause
Lack of a centralized, reusable HTTP client in the `online` package or `WappalyzerEngine`.

## Proposed Solution
- Create a global or engine-level `*http.Client`.
- Configure the `http.Transport` with increased `MaxIdleConns` and `MaxIdleConnsPerHost` (e.g., 500 and 100 respectively).
- Use this shared client in `FetchOnline`.

## Related Files
- `input/online/online.go`
- `cmd/root.go`

## Risk Level
Low

## Dependencies
None

## Expected Performance Impact
High. Dramatic reduction in latency for online scans and reduced CPU usage by avoiding repeated TLS handshakes.

## Status
done

## Implementation Notes
- Refactored `online.go` to use a singleton `http.Client`.
- Configured `http.Transport` with `MaxIdleConns: 500` and `MaxIdleConnsPerHost: 100`.
- Added `sync.Once` for thread-safe initialization.
