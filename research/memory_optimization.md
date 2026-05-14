# Research: Memory Optimization in Go

## 1. Object Pooling (`sync.Pool`)
- **Current Usage:** `OfflineInput` is already pooled.
- **Potential Extension:** Pool byte buffers for body reading or header maps if they are frequently allocated and discarded.
- **Warning:** Be careful with large buffers in `sync.Pool` as they might lead to high RSS if not managed correctly.

## 2. Allocation Profiling
- **Strategy:** Use `go test -bench . -benchmem` to identify high-allocation functions.
- **Optimization:** 
    - Pre-allocate slices/maps if the size is known (`make([]T, 0, capacity)`).
    - Avoid string concatenations in loops; use `strings.Builder`.
    - Use `[]byte` where possible to avoid `string` conversions.

## 3. String Interning
- **Benefit:** If many targets have identical technology names (e.g., "Apache", "PHP"), interning these strings can save significant memory.
- **Implementation:** Use a simple map or a specialized library for string interning.

## 4. Reducing Map Operations
- **Finding:** Maps are expensive in terms of hashing and potential bucketing.
- **Optimization:** In `handleResults`, the `tagMap` is cleared/re-used. If possible, use a more efficient data structure or pre-size the map.
