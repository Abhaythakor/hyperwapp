package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime" // Added for GOMAXPROCS
	"sync"

	"github.com/spf13/cobra"
	"hyperwapp/config"    // Added config package import
	"hyperwapp/detect"    // Added detect package import
	"hyperwapp/input"
	"hyperwapp/input/online" // Added input/online package import
	"hyperwapp/model"        // Added model package import for InputTypeOnline
	"hyperwapp/output"       // Added output package import
	"hyperwapp/progress"     // Added progress package import
	"hyperwapp/util"
)

var (
	offline       bool
	headersOnly   bool
	bodyOnly      bool
	auto          bool
	all           bool
	domain        bool
	outputFile    string
	outputFormat  string
	concurrency   int // Renamed from threads
	cpus          int // Added cpus for GOMAXPROCS
	timeout       int
	forceColor    bool
	disableColor  bool
	verbose       bool
	silent        bool // Added silent flag
	resume        bool
	update        bool
	showVersion   bool

	wappalyzerEngine detect.Engine
)

var resumeMgr *util.ResumeManager // Global resume manager instance

var rootCmd = &cobra.Command{
	Use:   "hyperwapp [input] [flags]",
	Short: "HyperWapp is a CLI reconnaissance utility",
	Long: `
   __  __                      _       __                
  / / / /_  ______  ___  _____| |     / /___ _____  ____ 
 / /_/ / / / / __ \/ _ \/ ___/| | /| / / __ \/ __ \/ __ \
/ __  / /_/ / /_/ /  __/ /    | |/ |/ / /_/ / /_/ / /_/ /
/_/ /_/\__, / .___/\___/_/     |__/|__/\__,_/ .___/ .___/ 
      /____/_/                             /_/   /_/      

A high-performance CLI tool to detect web technologies using Wappalyzer fingerprints.

Key Features:
- Powered by ProjectDiscovery's WappalyzerGo.
- Supports single URLs, URL lists, and piped input.
- Advanced Offline Mode: Recursively parses results from Katana, FFF, Raw HTTP dumps, or raw body files.
- Massively Scalable: Disk-backed streaming handles 10,000,000+ targets without memory issues.
- Resumable: Checkpoint system allows restarting interrupted scans instantly.
- Multiple Formats: CSV, JSON, TXT, Markdown, and real-time JSONL.
`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logging and color settings
		if forceColor {
			util.SetColorEnabled(true)
		} else if disableColor {
			util.SetColorEnabled(false)
		} else {
			util.SetColorEnabled(util.NewColorizer(false).Enabled)
		}

		if verbose {
			util.SetLogLevel(util.LevelDebug)
		} else if silent {
			util.SetLogLevel(util.LevelError)
		}

		if showVersion {
			fmt.Printf("HyperWapp Version: %s\n", config.Version)
			fmt.Printf("Wappalyzer Fingerprints: %s\n", detect.GetFingerprintsInfo())
			os.Exit(0)
		}

		if update {
			if err := detect.UpdateFingerprints(); err != nil {
				util.Fatal("Failed to update fingerprints: %v", err)
			}
			os.Exit(0)
		}

		util.Debug("PersistentPreRunE: Verbose flag set to: %t", verbose)

		// Set CPU limit (Parallelism)
		if cpus > 0 {
			util.Debug("Limiting CPU usage to %d cores", cpus)
			runtime.GOMAXPROCS(cpus)
		}

		// Initialize Wappalyzer engine
		var err error
		wappalyzerEngine, err = detect.NewWappalyzerEngine()
		if err != nil {
			util.Fatal("Failed to initialize Wappalyzer engine: %v", err)
		}

		// Initialize Resume Manager
		resumeMgr, err = util.NewResumeManager(".HyperWapp.resume", resume)
		if err != nil {
			util.Warn("Failed to initialize resume manager: %v", err)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var inputModeVal string // Use a distinct name to avoid shadowing inputMode in `setupWriter`
		var inputSource string
		
		if len(args) > 0 {
			inputSource = args[0]
		} else if isInputFromPipe() {
			inputSource = "-" // Indicate stdin
		}

		if inputSource == "" {
			util.Fatal("No input provided. Use 'HyperWapp [URL|FILE]' or pipe input.")
		}

		if all && domain {
			util.Fatal("Flags -all and -domain are mutually exclusive.")
		}

		var (
			tracker *progress.Tracker
			err     error
		)

		// --- Online/Offline mode logic to populate resultCh and initialize tracker ---
		var resultCh <-chan []model.Detection // Channel to receive detection batches from workers

		if offline {
			inputModeVal = "offline"
			tracker, resultCh = runOffline(inputSource)
		} else { // Online mode with concurrency
			inputModeVal = "online"
			tracker, resultCh = runOnline(inputSource)
		}

		// --- Process results from resultCh ---
		// Always initialize CLI writer
		cliWriter := output.NewCLIWriter(!disableColor)
		if domain {
			cliWriter.SetMode("domain")
		} else {
			cliWriter.SetMode("all")
		}

		// Optionally initialize file writer
		var fileWriter output.Writer
		if outputFile != "" {
			fileWriter, err = setupWriter(outputFormat, outputFile, !disableColor, inputModeVal, config.Version, resume)
			if err != nil {
				util.Fatal("Failed to set up file writer: %v", err)
			}
			if domain {
				fileWriter.SetMode("domain")
			} else {
				fileWriter.SetMode("all")
			}
		}

		// Stream detections
		for detections := range resultCh {
			// 1. Clear progress line
			tracker.Clear()

			// 2. Print to CLI
			if err := cliWriter.Write(detections); err != nil {
				util.Warn("Error writing to CLI: %v", err)
			}

			// 3. Write to File
			if fileWriter != nil {
				if err := fileWriter.Write(detections); err != nil {
					util.Warn("Error writing to file: %v", err)
				}
			}

			// 4. Restore progress line immediately
			tracker.Refresh()
		}

		tracker.Done() // Ensure tracker finishes

		// Finalize
		cliWriter.Close()
		if fileWriter != nil {
			fileWriter.Close()
		}
		resumeMgr.Cleanup()
	},
}

// processOnlineTarget fetches, detects, and enriches detections for a single online target.
func processOnlineTarget(target model.Target, timeout int) []model.Detection {
	util.Debug("  Scanning URL: %s", target.URL)
	headers, body, err := online.FetchOnline(target, timeout)
	if err != nil {
		util.Warn("Failed to fetch %s: %v", target.URL, err)
		return nil
	}

	detections, err := wappalyzerEngine.Detect(headers, body, model.SourceWappalyzer)
	if err != nil {
		util.Warn("Failed to detect technologies for %s: %v", target.URL, err)
		return nil
	}

	// Enrich detections with domain and URL from target
	for i := range detections {
		detections[i].Domain = target.Domain
		detections[i].URL = target.URL
	}
	return detections
}



// isInputFromPipe checks if the application is receiving input from a pipe.
func isInputFromPipe() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// A common way to detect piped input in Go is to check if it's not a character device (terminal).
	// This is more reliable than checking fi.Size() which can be 0 for pipes.
	return (fi.Mode() & os.ModeCharDevice) == 0
}

func runOffline(inputSource string) (*progress.Tracker, <-chan []model.Detection) {
	absInputSource, err := filepath.Abs(inputSource)
	if err != nil {
		util.Fatal("Error resolving absolute path for input: %v", err)
	}

	// Phase 1: Fast Discovery (Counting)
	tracker := progress.NewTracker(0, silent, !disableColor)
	var total uint32
	if resume && resumeMgr.TotalCount > 0 {
		total = resumeMgr.TotalCount
		util.Debug("Resuming scan with saved total: %d", total)
	} else {
		total, err = input.CountOffline(absInputSource, concurrency)
		if err != nil {
			util.Fatal("Error during discovery phase: %v", err)
		}
		resumeMgr.SaveTotal(total)
	}
	tracker.AddTotal(total)
	tracker.FinalizeTotal()

	// Phase 2: Scanning
	offlineInputCh, err := input.ParseOffline(absInputSource)
	if err != nil {
		util.Fatal("Error initializing offline parsing: %v", err)
	}

	offlineWorkerInputCh := make(chan model.OfflineInput)
	resultChWorker := make(chan []model.Detection)
	var wg sync.WaitGroup

	numWorkers := concurrency
	if numWorkers <= 0 {
		numWorkers = 1
	}

	// Discovery Loop
	go func() {
		defer close(offlineWorkerInputCh)
		for offInput := range offlineInputCh {
			offlineWorkerInputCh <- offInput
		}
	}()

	// Worker Pool
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for offInput := range offlineWorkerInputCh {
				id := offInput.URL
				if id == "" {
					id = offInput.Domain
				}

				if resumeMgr.IsCompleted(id) {
					tracker.Increment()
					continue
				}

				currentInputType := model.InputTypeOffline
				if headersOnly {
					currentInputType = model.SourceHeadersOnly
				} else if bodyOnly {
					currentInputType = model.SourceBodyOnly
				}

				detections, err := wappalyzerEngine.Detect(offInput.Headers, offInput.Body, currentInputType)
				if err != nil {
					util.Warn("Failed to detect technologies for offline input (Domain: %s, URL: %s): %v", offInput.Domain, offInput.URL, err)
					tracker.IncrementError()
					continue
				}

				for i := range detections {
					detections[i].Domain = offInput.Domain
					detections[i].URL = offInput.URL
				}
				resultChWorker <- detections
				resumeMgr.MarkCompleted(id)
				tracker.IncrementSuccess()
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultChWorker)
	}()

	return tracker, resultChWorker
}

func runOnline(inputSource string) (*progress.Tracker, <-chan []model.Detection) {
	targets, err := input.ResolveInput(inputSource, false) // Explicitly false for online mode
	if err != nil {
		util.Fatal("Error resolving input: %v", err)
	}

	util.Debug("Resolved %d targets for online scanning:", len(targets))

	tracker := progress.NewTracker(uint32(len(targets)), silent, !disableColor)
	targetCh := make(chan model.Target)
	resultChWorker := make(chan []model.Detection)
	var wg sync.WaitGroup

	numWorkers := concurrency
	if numWorkers <= 0 {
		numWorkers = 1
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for target := range targetCh {
				if resumeMgr.IsCompleted(target.URL) {
					tracker.IncrementSuccess() // Count as success since it's already done
					continue
				}
				detections := processOnlineTarget(target, timeout)
				if detections == nil {
					tracker.IncrementError()
					continue
				}
				resultChWorker <- detections
				resumeMgr.MarkCompleted(target.URL)
				tracker.IncrementSuccess()
			}
		}()
	}

	go func() {
		for _, target := range targets {
			targetCh <- target
		}
		close(targetCh)
	}()

	go func() {
		wg.Wait()
		close(resultChWorker)
	}()

	return tracker, resultChWorker
}

func setupWriter(outputFormat, outputFile string, colorize bool, inputType string, version string, resume bool) (output.Writer, error) {
	switch outputFormat {
	case "csv":
		return output.NewCSVWriter(outputFile, resume)
	case "json":
		return output.NewJSONWriter(outputFile, inputType, version)
	case "jsonl":
		return output.NewJSONLWriter(outputFile, resume)
	case "txt":
		return output.NewTXTWriter(outputFile)
	case "md":
		return output.NewMDWriter(outputFile)
	default:
		return nil, fmt.Errorf("unsupported output format for file: %s", outputFormat)
	}
}


func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&offline, "offline", false, "Offline mode: recursively parse directory structure (Katana, FFF, etc.)")
	rootCmd.PersistentFlags().BoolVar(&headersOnly, "headers-only", false, "Detect technologies using HTTP headers only")
	rootCmd.PersistentFlags().BoolVar(&bodyOnly, "body-only", false, "Detect technologies using HTTP body only")
	rootCmd.PersistentFlags().BoolVar(&auto, "auto", true, "Detect using both headers and body (default)")

	rootCmd.PersistentFlags().BoolVar(&all, "all", false, "Output results per URL (default)")
	rootCmd.PersistentFlags().BoolVar(&domain, "domain", false, "Aggregate and output results per unique domain")

	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "Write output to specified file")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "cli", "Output format: csv, json, jsonl, txt, md")

	rootCmd.PersistentFlags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of concurrent workers (goroutines)")
	rootCmd.PersistentFlags().IntVarP(&concurrency, "threads", "t", 10, "Alias for --concurrency")
	rootCmd.PersistentFlags().IntVar(&cpus, "cpus", 0, "Limit number of physical CPU cores to use (GOMAXPROCS)")
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 10, "HTTP timeout in seconds for online scanning")
	
	rootCmd.PersistentFlags().BoolVar(&forceColor, "color", false, "Force colored CLI output")
	rootCmd.PersistentFlags().BoolVar(&disableColor, "no-color", false, "Disable colored CLI output")
	rootCmd.PersistentFlags().BoolVar(&disableColor, "mono", false, "Alias for --no-color")
	rootCmd.PersistentFlags().BoolVar(&silent, "silent", false, "Display results only (suppress tracker and info logs)")
	rootCmd.PersistentFlags().BoolVar(&resume, "resume", false, "Resume an interrupted scan using .HyperWapp.resume checkpoint")
	rootCmd.PersistentFlags().BoolVar(&update, "update", false, "Update Wappalyzer fingerprints from ProjectDiscovery GitHub")
	rootCmd.PersistentFlags().BoolVar(&showVersion, "version", false, "Show tool version and fingerprints information")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose debug logging")
}

