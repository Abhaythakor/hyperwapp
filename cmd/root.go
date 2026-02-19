package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/Abhaythakor/hyperwapp/config"
	"github.com/Abhaythakor/hyperwapp/detect"
	"github.com/Abhaythakor/hyperwapp/input"
	"github.com/Abhaythakor/hyperwapp/input/online"
	"github.com/Abhaythakor/hyperwapp/model"
	"github.com/Abhaythakor/hyperwapp/output"
	"github.com/Abhaythakor/hyperwapp/progress"
	"github.com/Abhaythakor/hyperwapp/util"
	"github.com/spf13/cobra"
)

var (
	offline      bool
	headersOnly  bool
	bodyOnly     bool
	auto         bool
	all          bool
	domain       bool
	outputFile   string
	outputFormat string
	url          string
	urlList      string
	concurrency  int
	cpus         int
	timeout      int
	forceColor   bool
	disableColor bool
	verbose      bool
	silent       bool
	resume       bool
	update       bool
	showVersion  bool

	wappalyzerEngine detect.Engine
)

var resumeMgr *util.ResumeManager

var rootCmd = &cobra.Command{
	Use:   "hyperwapp [flags]",
	Short: "HyperWapp is a CLI reconnaissance utility",
	Long: `
   __  __                      _       __                
  / / / /_  ______  ___  _____| |     / /___ _____  ____ 
 / /_/ / / / / __ \/ _ \/ ___/| | /| / / __ \/ __ \/ __ \
/ __  / /_/ / /_/ /  __/ /    | |/ |/ / /_/ / /_/ / /_/ /
/_/ /_/\__, / .___/\___/_/     |__/|__/\__,_/ .___/ .___/ 
      /____/_/                             /_/   /_/      

A high-performance, massively scalable CLI tool to detect web technologies using Wappalyzer fingerprints.

EXAMPLES:
  # Online scan (Single URL)
  hyperwapp -u https://example.com
  
  # Online scan (URL list)
  hyperwapp -l urls.txt -c 50
  
  # Offline scan (Recursive directory)
  hyperwapp -offline ./responses/ -f jsonl -o results.jsonl
  
  # Resume an interrupted scan
  hyperwapp -l massive_list.txt --resume

  # Piping input
  subfinder -d airbnb.com | httpx | hyperwapp -silent

ADDITIONAL INFO:
  - Powered by ProjectDiscovery's WappalyzerGo.
  - Disk-backed streaming handles 10,000,000+ targets with low RAM.
  - Real-time JSONL output and checkpoint system for reliability.
`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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

		if cpus > 0 {
			runtime.GOMAXPROCS(cpus)
		}

		var err error
		wappalyzerEngine, err = detect.NewWappalyzerEngine()
		if err != nil {
			util.Fatal("Failed to initialize Wappalyzer engine: %v", err)
		}

		resumeMgr, err = util.NewResumeManager(".HyperWapp.resume", resume)
		if err != nil {
			util.Warn("Failed to initialize resume manager: %v", err)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var inputModeVal string
		var inputSource string

		if urlList != "" {
			inputSource = urlList
		} else if url != "" {
			inputSource = url
		} else if len(args) > 0 {
			inputSource = args[0]
		} else if isInputFromPipe() {
			inputSource = "-"
		}

		if inputSource == "" {
			util.Fatal("No input provided. Use -u, -l, positional argument, or pipe input.")
		}

		if all && domain {
			util.Fatal("Flags -all and -domain are mutually exclusive.")
		}

		if offline {
			inputModeVal = "offline"
			tracker, resultCh := runOffline(inputSource)
			handleResults(resultCh, tracker, inputModeVal)
		} else {
			inputModeVal = "online"
			tracker, resultCh := runOnline(inputSource)
			handleResults(resultCh, tracker, inputModeVal)
		}
	},
}

func handleResults(resultCh <-chan []model.Detection, tracker *progress.Tracker, inputModeVal string) {
	cliWriter := output.NewCLIWriter(!disableColor)
	if domain {
		cliWriter.SetMode("domain")
	} else {
		cliWriter.SetMode("all")
	}

	var fileWriter output.Writer
	if outputFile != "" {
		var err error
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

	for detections := range resultCh {
		tracker.Clear()
		if err := cliWriter.Write(detections); err != nil {
			util.Warn("Error writing to CLI: %v", err)
		}
		if fileWriter != nil {
			if err := fileWriter.Write(detections); err != nil {
				util.Warn("Error writing to file: %v", err)
			}
		}
		tracker.Refresh()
	}

	tracker.Done()
	cliWriter.Close()
	if fileWriter != nil {
		fileWriter.Close()
	}
	resumeMgr.Cleanup()
}

func processOnlineTarget(target model.Target, timeout int) []model.Detection {
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

	for i := range detections {
		detections[i].Domain = target.Domain
		detections[i].URL = target.URL
	}
	return detections
}

func isInputFromPipe() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) == 0
}

func runOffline(inputSource string) (*progress.Tracker, <-chan []model.Detection) {
	absInputSource, err := filepath.Abs(inputSource)
	if err != nil {
		util.Fatal("Error resolving absolute path for input: %v", err)
	}

	tracker := progress.NewTracker(0, silent, !disableColor)
	var total uint32
	if resume && resumeMgr.TotalCount > 0 {
		total = resumeMgr.TotalCount
	} else {
		total, err = input.CountOffline(absInputSource, concurrency)
		if err != nil {
			util.Fatal("Error during discovery phase: %v", err)
		}
		resumeMgr.SaveTotal(total)
	}
	tracker.AddTotal(total)
	tracker.FinalizeTotal()

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

	go func() {
		defer close(offlineWorkerInputCh)
		for offInput := range offlineInputCh {
			offlineWorkerInputCh <- offInput
		}
	}()

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
					tracker.IncrementSuccess()
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
	targets, err := input.ResolveInput(inputSource, false)
	if err != nil {
		util.Fatal("Error resolving input: %v", err)
	}

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
					tracker.IncrementSuccess()
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
	// Input Group
	rootCmd.PersistentFlags().StringVarP(&url, "url", "u", "", "Single URL to scan")
	rootCmd.PersistentFlags().StringVarP(&urlList, "list", "l", "", "File containing list of URLs to scan")
	rootCmd.PersistentFlags().BoolVar(&offline, "offline", false, "Offline mode: recursively parse directory structure (Katana, FFF, etc.)")

	// Detection Strategy Group
	rootCmd.PersistentFlags().BoolVar(&headersOnly, "headers-only", false, "Detect technologies using HTTP headers only")
	rootCmd.PersistentFlags().BoolVar(&bodyOnly, "body-only", false, "Detect technologies using HTTP body only")
	rootCmd.PersistentFlags().BoolVar(&auto, "auto", true, "Detect using both headers and body (default)")

	// Output Mode Group
	rootCmd.PersistentFlags().BoolVar(&all, "all", false, "Output results per URL (default)")
	rootCmd.PersistentFlags().BoolVar(&domain, "domain", false, "Aggregate and output results per unique domain")

	// Export Group
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "Write output to specified file")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "cli", "Output format: csv, json, jsonl, txt, md")

	// Performance Group
	rootCmd.PersistentFlags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of concurrent workers (goroutines)")
	rootCmd.PersistentFlags().IntVarP(&concurrency, "threads", "t", 10, "Alias for --concurrency")
	rootCmd.PersistentFlags().IntVar(&cpus, "cpus", 0, "Limit number of physical CPU cores to use (GOMAXPROCS)")
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 10, "HTTP timeout in seconds for online scanning")

	// UI & Debug Group
	rootCmd.PersistentFlags().BoolVar(&forceColor, "color", false, "Force colored CLI output")
	rootCmd.PersistentFlags().BoolVar(&disableColor, "no-color", false, "Disable colored CLI output")
	rootCmd.PersistentFlags().BoolVar(&disableColor, "mono", false, "Alias for --no-color")
	rootCmd.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "Display results only (suppress tracker and info logs)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose debug logging")

	// Utility Group
	rootCmd.PersistentFlags().BoolVar(&resume, "resume", false, "Resume an interrupted scan using .HyperWapp.resume checkpoint")
	rootCmd.PersistentFlags().BoolVar(&update, "update", false, "Update Wappalyzer fingerprints from ProjectDiscovery GitHub")
	rootCmd.PersistentFlags().BoolVar(&showVersion, "version", false, "Show tool version and fingerprints information")
}
