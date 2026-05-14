# Task 005: Memory Optimization: Reducing Allocations

## Problem Description
High GC pressure due to frequent small allocations in the hot path.

## Root Cause
Inefficient use of strings, maps, and byte slices.

## Proposed Solution
- Use `strings.Builder` for any dynamic string construction.
- Pre-allocate slices/maps where sizes are predictable.
- Investigate pooling of `headers` maps.
- Implement string interning for common technology names.

## Related Files
- `model/detection.go`
- `detect/engine.go`
- `input/fff/fff.go` (and other parsers)

## Risk Level
Low

## Dependencies
None

## Expected Performance Impact
Medium. Reduced GC pauses and overall memory footprint.

## Status
done

## Implementation Notes
- **Detection Engine:** Pre-allocated the `detections` slice in `detect/engine.go` to reduce allocations in the hot loop.
- **Header Parsing:** Optimized `fff`, `katana`, and `raw` parsers to reuse the pooled `Headers` map from `OfflineInput` instead of allocating a new map for every target.
- **Discovery Phase:** Added a `sync.Pool` for the 1MB line-counting buffer in `input/offline.go` to reduce memory pressure during the initial target counting phase.
