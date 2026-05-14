# Task 004: Parallel Result Handling & Tag Mapping

## Problem Description
`handleResults` is a single-threaded bottleneck that performs Nuclei mapping and multiple writes.

## Root Cause
Sequential implementation of result processing.

## Proposed Solution
- De-couple Nuclei mapping from the output loop.
- Use a pool of goroutines for mapping if necessary, or simply ensure that mapping is done by workers *before* sending to `resultCh`.
- Ensure writers are protected by mutexes (done for JSONL, check others).

## Related Files
- `cmd/root.go`
- `output/`

## Risk Level
Medium

## Dependencies
None

## Expected Performance Impact
Medium. Prevents backpressure from output I/O and CPU-bound mapping.

## Status
done

## Implementation Notes
- Moved `detect.MapToNucleiTag` from the single-threaded `handleResults` goroutine into the parallel worker pools in `runOnline`, `runOffline`, and `runProxy`.
- `handleResults` now only performs a lightweight unique tag collection for the final summary.
- This de-couples the CPU-intensive mapping (which involves regex) from the output I/O loop, preventing backpressure.
