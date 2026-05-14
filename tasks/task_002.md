# Task 002: Buffered Channels in Online Mode

## Problem Description
`targetCh` and `resultChWorker` in `runOnline` are unbuffered.

## Root Cause
Simple channel initialization without specifying capacity.

## Proposed Solution
- Add reasonable buffering to `targetCh` (e.g., 1000) and `resultChWorker` (e.g., 2000).
- Match the buffering patterns used in `runOffline`.

## Related Files
- `cmd/root.go`

## Risk Level
Low

## Dependencies
None

## Expected Performance Impact
Medium. Reduced worker blocking and smoother pipeline flow.

## Status
done

## Implementation Notes
- Added buffer of 1000 to `targetCh`.
- Added buffer of 2000 to `resultChWorker`.
