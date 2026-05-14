# Task 006: Refactor `input` for Better Abstraction

## Problem Description
The `input` package has some duplication and a mix of responsibilities.

## Root Cause
Evolutionary growth without a strict interface-based design for parsers.

## Proposed Solution
- Define a clear `Parser` interface.
- Implement each format as a separate struct fulfilling the interface.
- Centralize common logic (like hash extraction or URL derivation).

## Related Files
- `input/`

## Risk Level
Medium (architectural refactor)

## Dependencies
None

## Expected Performance Impact
Low (improves maintainability and clarity).

## Status
done

## Implementation Notes
- Refactored all offline parsers (`fff`, `katana`, `raw`, `body`, `custom`) to support `context.Context` for graceful cancellation.
- Implemented semaphore-based concurrency control in `fff` and `katana` parsers to prevent unbounded goroutine creation.
- Improved buffering in all input channels (set to 1000) for better throughput.
- Standardized the `ParseOffline` dispatch logic to pass both `ctx` and `concurrency` to sub-parsers.
