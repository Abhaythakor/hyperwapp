# Checkpoint & Resume System ⏯️

HyperWapp includes a robust resume system designed specifically for long-running scans of 1,000,000+ targets.

---

## 1. How it Works

When you enable the `--resume` flag, HyperWapp creates a hidden file named `.hyperwapp.resume` in your current working directory.

This file serves two purposes:
1.  **Metadata Storage:** It saves the total count of discovered targets from the "Discovery Phase."
2.  **Activity Log:** Every time a worker successfully processes a target, its unique ID (URL or File Path) is appended to this log.

---

## 2. Using Resume

If your scan is interrupted (e.g., by a system crash or manually pressing `Ctrl+C`), simply run the exact same command again with the `--resume` flag.

```bash
# Interrupted scan
./hyperwapp -offline data/ -o results.jsonl --resume

# Restarting
./hyperwapp -offline data/ -o results.jsonl --resume
```

### What happens on restart:
*   **Instant Start:** HyperWapp reads the total count from the log, skipping the slow "Discovery Phase" entirely.
*   **Target Skipping:** The tool loads the IDs of completed targets into a lookup table. Any target already in the log is skipped without being scanned again.
*   **Append Output:** HyperWapp automatically detects that you are resuming and **appends** new results to your existing output file instead of overwriting it.

---

## 3. Best Practices

### Cleanup
When a scan completes 100% successfully, HyperWapp automatically deletes the `.hyperwapp.resume` file to keep your directory clean.

### One Resume File per Directory
Since the checkpoint file is named `.hyperwapp.resume`, avoid running multiple different scans in the same folder at the same time if they both use `--resume`. 

---

## 4. Manual Inspection
The `.hyperwapp.resume` file is a plain text file. You can peek inside to see the progress:
```bash
# See total and first few items
head -n 5 .hyperwapp.resume
```
