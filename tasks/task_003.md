# Task 003: Context Propagation & Graceful Shutdown

## Problem Description
Cancellation via Ctrl+C is partially handled in `Run` but not propagated down to the fetchers or parsers.

## Root Cause
Components do not accept `context.Context` in their signatures.

## Proposed Solution
- Update `FetchOnline`, `ParseOffline`, and worker loops to accept and respect `context.Context`.
- Ensure all loops check `ctx.Done()`.

## Related Files
- `cmd/root.go`
- `input/online/online.go`
- `input/offline.go`

## Risk Level
Medium (requires signature changes)

## Dependencies
None

## Expected Performance Impact
Low (improves reliability and shutdown behavior).

## Status
done

## Implementation Notes
- Updated `runOnline`, `runOffline`, and `runProxy` to accept `context.Context`.
- Propagated `ctx` to `FetchOnline` and `input.ParseOffline`.
- Added `select` blocks with `ctx.Done()` in all major producer/consumer loops to ensure immediate and graceful shutdown.
