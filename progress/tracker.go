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
	lastUpdate time.Time
	quiet      bool
	enabled    bool
	color      *util.Colorizer
	started    bool
	finalized  atomic.Bool // Tracks if discovery is finished
	stopChan   chan struct{}
	isLogMode  bool // True for Termux or non-interactive terminals
	lastLog    time.Time
}

// NewTracker creates a new progress tracker.
func NewTracker(total uint32, quiet, colorize bool) *Tracker {
	t := &Tracker{
		startTime:  time.Now(),
		lastUpdate: time.Now(),
		lastLog:    time.Now(),
		quiet:      quiet,
		enabled:    !quiet,
		color:      util.NewColorizer(colorize),
		stopChan:   make(chan struct{}),
		isLogMode:  os.Getenv("TERMUX_VERSION") != "" || os.Getenv("TERM") == "dumb",
	}
	t.total.Store(total)
	if total > 0 {
		t.started = true
		t.finalized.Store(true) // If total is known at start, it's finalized
		t.printProgress(true)
	}

	if t.enabled {
		go t.refreshLoop()
	}

	return t
}

func (t *Tracker) refreshLoop() {
	ticker := time.NewTicker(250 * time.Millisecond) // Refresh UI at 4Hz (Saves CPU)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if t.started {
				t.printProgress(false)
			}
		case <-t.stopChan:
			return
		}
	}
}

// FinalizeTotal marks the discovery phase as complete.
func (t *Tracker) FinalizeTotal() {
	if !t.enabled {
		return
	}
	t.finalized.Store(true)
	t.startTime = time.Now()           // Reset timer for scanning phase
	if t.isLogMode {
		fmt.Fprintf(os.Stderr, "[+] Discovery Done.\n")
	} else {
		fmt.Fprintf(os.Stderr, " Done.\n")
	}
}

// IncrementSuccess records a successful scan.
func (t *Tracker) IncrementSuccess() {
	if !t.enabled {
		return
	}
	t.success.Add(1)
	t.completed.Add(1)
}

// IncrementError records a failed target.
func (t *Tracker) IncrementError() {
	if !t.enabled {
		return
	}
	t.errors.Add(1)
	t.completed.Add(1)
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
	}
}

// Increment increments the completed count and prints progress if enabled.
func (t *Tracker) Increment() {
	if !t.enabled {
		return
	}
	t.completed.Add(1)
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
	close(t.stopChan) // Stop refresh loop

	// Clear the progress line before printing final summary.
	if t.started && !t.isLogMode {
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
	if t.enabled && !t.isLogMode {
		fmt.Fprintf(os.Stderr, "\r\033[2K") // Clear the line
	}
}

// printProgress prints the current progress on the same line with throttling.
func (t *Tracker) printProgress(force bool) {
	if !t.enabled || !t.started {
		return
	}

	// Throttle UI updates to save CPU power (Max 5 updates per second)
	if !force && time.Since(t.lastUpdate) < (200 * time.Millisecond) {
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
		progressLine = fmt.Sprintf("[+] Discovering: %s...", t.color.Yellow(fmt.Sprintf("%d", total)))
	} else {
		// Scanning Phase UI
		rps := 0.0
		if elapsed.Seconds() > 0 {
			rps = float64(completed) / elapsed.Seconds()
		}

		percent := "0.0%"
		if total > 0 {
			p := float64(completed) / float64(total) * 100
			percent = fmt.Sprintf("%.1f%%", p)
		}

		if t.isLogMode {
			// Clean log-style output for Termux
			progressLine = fmt.Sprintf("[+] %s | Processed: %d/%d | Success: %d | Errors: %d | %.1f/s",
				percent, completed, total, success, errors, rps)
		} else {
			// Compact PC version
			progressLine = fmt.Sprintf("[+] %s | %d/%d | S:%s | E:%s | %.0f/s | %s",
				t.color.Cyan(percent),
				completed, total,
				t.color.Green(fmt.Sprintf("%d", success)),
				t.color.Red(fmt.Sprintf("%d", errors)),
				rps,
				elapsed.Round(time.Second))
		}
	}

	if t.isLogMode {
		// Print every 20 units OR every 2 seconds to keep it interactive
		now := time.Now()
		if force || completed%20 == 0 || completed == total || now.Sub(t.lastLog) > 2*time.Second {
			fmt.Fprintf(os.Stderr, "%s\n", progressLine)
			t.lastLog = now
		}
	} else {
		// ANSI: Clear entire line (2K) and return to start (\r)
		fmt.Fprintf(os.Stderr, "\r\033[2K%s", progressLine)
	}
}
