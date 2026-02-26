package progress

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/Abhaythakor/hyperwapp/util"
)

// Tracker manages and displays progress updates.
type Tracker struct {
	total      atomic.Uint32
	completed  atomic.Uint32
	success    atomic.Uint32 // Tracks targets with detections
	errors     atomic.Uint32 // Tracks timeouts/network errors
	startTime  time.Time
	lastUpdate time.Time     // New: For throttling
	quiet      bool
	enabled    bool
	color      *util.Colorizer
	started    bool
	finalized  atomic.Bool // Tracks if discovery is finished
}

// NewTracker creates a new progress tracker.
func NewTracker(total uint32, quiet, colorize bool) *Tracker {
	t := &Tracker{
		startTime:  time.Now(),
		lastUpdate: time.Now(),
		quiet:      quiet,
		enabled:    !quiet,
		color:      util.NewColorizer(colorize),
	}
	t.total.Store(total)
	if total > 0 {
		t.started = true
		t.finalized.Store(true) // If total is known at start, it's finalized
		t.printProgress(true)   // Force print first update
	}
	return t
}

// FinalizeTotal marks the discovery phase as complete.
func (t *Tracker) FinalizeTotal() {
	if !t.enabled {
		return
	}
	t.finalized.Store(true)
	t.startTime = time.Now()           // Reset timer for scanning phase
	fmt.Fprintf(os.Stderr, " Done.\n") // Finish the discovery line and move to next
}

// IncrementSuccess records a successful scan.
func (t *Tracker) IncrementSuccess() {
	if !t.enabled {
		return
	}
	t.success.Add(1)
	t.completed.Add(1)
	t.printProgress(false)
}

// IncrementError records a failed target.
func (t *Tracker) IncrementError() {
	if !t.enabled {
		return
	}
	t.errors.Add(1)
	t.completed.Add(1)
	t.printProgress(false)
}

// AddTotal atomically adds to the total count.
func (t *Tracker) AddTotal(count uint32) {
	if !t.enabled {
		return
	}
	t.total.Add(count)
	if !t.started && t.total.Load() > 0 {
		t.started = true
		t.printProgress(true) // Force print initial state
	} else if t.started && !t.finalized.Load() {
		t.printProgress(false)
	}
}

// Increment increments the completed count and prints progress if enabled.
func (t *Tracker) Increment() {
	if !t.enabled {
		return
	}

	t.completed.Add(1)

	// Only print if the tracker has actually started (i.e., total is known and > 0)
	// and if we have completed items, to avoid showing Completed:0 initially.
	if t.started {
		t.printProgress(false)
	}
}

// Refresh re-prints the current progress line.
func (t *Tracker) Refresh() {
	t.printProgress(true) // Force refresh
}

// Done marks the tracker as complete and prints final summary.
func (t *Tracker) Done() {
	if !t.enabled {
		return
	}
	// Clear the progress line before printing final summary.
	if t.started {
		t.Clear()
	}

	completed := t.completed.Load()
	success := t.success.Load()
	errors := t.errors.Load()
	elapsed := time.Since(t.startTime).Round(time.Second)

	// Final summary in the same style as scanning
	fmt.Fprintf(os.Stderr, "[+] Scan Finished: Processed %d targets in %s (Success: %d, Errors: %d)\n",
		completed, elapsed, success, errors)
}

// Clear clears the progress line.
func (t *Tracker) Clear() {
	if t.enabled {
		fmt.Fprintf(os.Stderr, "\033[2K\r") // Clear the line
	}
}

// printProgress prints the current progress on the same line with throttling.
func (t *Tracker) printProgress(force bool) {
	if !t.enabled || !t.started {
		return
	}

	// Throttle to 15 updates per second
	if !force && time.Since(t.lastUpdate) < (time.Second/15) {
		return
	}
	t.lastUpdate = time.Now()

	total := t.total.Load()
	completed := t.completed.Load()
	success := t.success.Load()
	errors := t.errors.Load()
	finalized := t.finalized.Load()
	elapsed := time.Since(t.startTime)

	var progressLine string
	if !finalized {
		// Discovery Phase UI
		progressLine = fmt.Sprintf("[+] Discovering targets: %s...", t.color.Yellow(fmt.Sprintf("%d", total)))
	} else {
		// Scanning Phase UI - Persistent Footer
		rps := 0.0
		if elapsed.Seconds() > 0 {
			rps = float64(completed) / elapsed.Seconds()
		}

		// Calculate percentage if total is known
		percent := ""
		if total > 0 {
			p := float64(completed) / float64(total) * 100
			percent = fmt.Sprintf("%.1f%%", p)
		}

		progressLine = fmt.Sprintf("[+] %s | Processed: %d/%d | Success: %s | Errors: %s | Speed: %.2f/s | Time: %s",
			t.color.Cyan(percent),
			completed, total,
			t.color.Green(fmt.Sprintf("%d", success)),
			t.color.Red(fmt.Sprintf("%d", errors)),
			rps,
			elapsed.Round(time.Second))
	}

	// ANSI escape code to clear the current line (2K) and move cursor to beginning (\r)
	fmt.Fprintf(os.Stderr, "\033[2K\r%s", progressLine)
}
