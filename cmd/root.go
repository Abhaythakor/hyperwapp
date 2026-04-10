package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/Abhaythakor/hyperwapp/config"
	"github.com/Abhaythakor/hyperwapp/detect"
	"github.com/Abhaythakor/hyperwapp/input"
	"github.com/Abhaythakor/hyperwapp/input/custom"
	"github.com/Abhaythakor/hyperwapp/input/online"
	"github.com/Abhaythakor/hyperwapp/input/proxy" // Added proxy import
	"github.com/Abhaythakor/hyperwapp/model"
	"github.com/Abhaythakor/hyperwapp/output"
	"github.com/Abhaythakor/hyperwapp/progress"
	"github.com/Abhaythakor/hyperwapp/util"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	offline      bool
	headersOnly  bool
	bodyOnly     bool
	auto         bool
	all          bool
	domain       bool
	outputFile     string
	outputFormat   string
	url            string
	urlList        string
	proxyAddr      string // Added for proxy mode
	inputConfigPath string
	concurrency    int
	cpus           int
	timeout      int
	forceColor   bool
	disableColor bool
	verbose      bool
	silent       bool
	resume       bool
	update       bool
	showVersion  bool
	showNuclei   bool // Added for nuclei bridge

	wappalyzerEngine *detect.WappalyzerEngine
)

var resumeMgr *util.ResumeManager

var rootCmd = &cobra.Command{
	Use:   "hyperwapp [flags] [input]",
	Short: "HyperWapp is a CLI reconnaissance utility",
	Args:  cobra.ArbitraryArgs,
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
			// Update Fingerprints
			if err := detect.UpdateFingerprints(); err != nil {
				util.Warn("Failed to update fingerprints: %v", err)
			}

			// Update Binary via Go Install (Bypass cache with GOPROXY=direct)
			util.Info("Updating HyperWapp binary via go install...")
			cmd := exec.Command("go", "install", "github.com/Abhaythakor/hyperwapp@latest")
			cmd.Env = append(os.Environ(), "GOPROXY=direct") // Force direct download from GitHub
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				util.Fatal("Failed to update HyperWapp: %v", err)
			}

			// Local Rebuild Logic: Check if we are running from a local development folder
			exePath, err := os.Executable()
			if err == nil {
				realExePath, _ := filepath.EvalSymlinks(exePath)
				absExeDir := filepath.Dir(realExePath)
				cwd, _ := os.Getwd()
				absCwd, _ := filepath.Abs(cwd)

				if absExeDir == absCwd {
					util.Info("Local binary detected at %s. Rebuilding...", realExePath)
					buildCmd := exec.Command("go", "build", "-o", filepath.Base(realExePath), "main.go")
					if err := buildCmd.Run(); err == nil {
						util.Info("Local binary updated successfully!")
					} else {
						util.Warn("Local rebuild failed: %v", err)
					}
				}
			}

			util.Info("HyperWapp updated successfully!")
			os.Exit(0)
		}

		// Background version check
		if !silent && !showVersion {
			go util.CheckForUpdates(config.Version)
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
		} else if proxyAddr != "" {
			inputSource = proxyAddr
		} else if len(args) > 0 {
			inputSource = args[0]
		} else if isInputFromPipe() {
			inputSource = "-"
		}

		if inputSource == "" {
			util.Fatal("No input provided. Use -u, -l, -proxy, positional argument, or pipe input.")
		}

		if all && domain {
			util.Fatal("Flags -all and -domain are mutually exclusive.")
		}

		var (
			tracker  *progress.Tracker
			resultCh <-chan []model.Detection
		)

		// Handle Ctrl+C for graceful shutdown and buffer flushing
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		if proxyAddr != "" {
			inputModeVal = "proxy"
			tracker, resultCh = runProxy(proxyAddr, wappalyzerEngine)
		} else if offline {
			inputModeVal = "offline"
			tracker, resultCh = runOffline(inputSource, wappalyzerEngine)
		} else {
			inputModeVal = "online"
			tracker, resultCh = runOnline(inputSource, wappalyzerEngine)
		}

		// Run result handler in the background
		done := make(chan struct{})
		go func() {
			handleResults(resultCh, tracker, inputModeVal)
			close(done)
		}()

		// Wait for either the scan to finish naturally or a Ctrl+C
		select {
		case <-ctx.Done():
			util.Info("Interrupt received, shutting down gracefully...")
		case <-done:
			// Scan finished naturally
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

	var allNucleiTags []string
	tagMap := make(map[string]struct{})

	// High-speed result processing loop
	for detections := range resultCh {
		if detections == nil {
			continue
		}

		// Map to Nuclei Tags
		for i := range detections {
			tag := detect.MapToNucleiTag(detections[i].Technology)
			if tag != "" {
				detections[i].NucleiTags = []string{tag}
				if _, exists := tagMap[tag]; !exists {
					tagMap[tag] = struct{}{}
					allNucleiTags = append(allNucleiTags, tag)
				}
			}
		}

		// Only print to CLI if NOT silent and we have technologies
		if !silent && len(detections) > 0 {
			tracker.Clear()
			if err := cliWriter.Write(detections); err != nil {
				util.Warn("Error writing to CLI: %v", err)
			}
		}

		if fileWriter != nil {
			if err := fileWriter.Write(detections); err != nil {
				util.Warn("Error writing to file: %v", err)
			}
		}
	}

	tracker.Done()

	// If --nuclei is set, print the bridge summary
	if showNuclei && len(allNucleiTags) > 0 {
		color := util.NewColorizer(!disableColor)
		fmt.Printf("\n[+] %s: %s\n", color.Cyan("Discovered Nuclei Tags"), strings.Join(allNucleiTags, ", "))
		fmt.Printf("[>] %s: nuclei -l targets.txt -tags %s\n\n", color.Yellow("Run Nuclei"), strings.Join(allNucleiTags, ","))
	}

	cliWriter.Close()
	if fileWriter != nil {
		fileWriter.Close()
	}
	resumeMgr.Cleanup()
}

func isInputFromPipe() bool {
	return !term.IsTerminal(int(os.Stdin.Fd()))
}

func runProxy(addr string, engine *detect.WappalyzerEngine) (*progress.Tracker, <-chan []model.Detection) {
	tracker := progress.NewTracker(0, silent, !disableColor)
	
	// Create channels
	proxyInputCh := make(chan model.OfflineInput, 100)
	resultChWorker := make(chan []model.Detection, 100)
	
	// Start Proxy Server
	go func() {
		if err := proxy.StartProxy(addr, proxyInputCh); err != nil {
			util.Fatal("Proxy error: %v", err)
		}
	}()

	// Start Workers (similar to offline but processing live proxy data)
	var wg sync.WaitGroup
	numWorkers := concurrency
	if numWorkers <= 0 {
		numWorkers = 1
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for input := range proxyInputCh {
				tracker.AddTotal(1) // Increment total as requests come in

				currentInputType := model.InputTypeOffline
				if headersOnly {
					currentInputType = model.SourceHeadersOnly
				} else if bodyOnly {
					currentInputType = model.SourceBodyOnly
				}

				detections, err := engine.Detect(input.Headers, input.Body, currentInputType)
				if err != nil {
					util.Warn("Failed to detect technologies for proxy input %s: %v", input.URL, err)
					tracker.IncrementError()
					continue
				}

				for i := range detections {
					detections[i].Domain = input.Domain
					detections[i].URL = input.URL
				}
				resultChWorker <- detections
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

func runOffline(inputSource string, engine *detect.WappalyzerEngine) (*progress.Tracker, <-chan []model.Detection) {
	absInputSource, err := filepath.Abs(inputSource)
	if err != nil {
		util.Fatal("Error resolving absolute path for input: %v", err)
	}

	// Load custom input config if provided
	var customCfg *custom.CompiledConfig
	if inputConfigPath != "" {
		customCfg, err = custom.LoadConfig(inputConfigPath)
		if err != nil {
			util.Fatal("Error loading input config: %v", err)
		}
	}

	tracker := progress.NewTracker(0, silent, !disableColor)
	var total uint32
	if resume && resumeMgr.TotalCount > 0 {
		total = resumeMgr.TotalCount
	} else {
		total, err = input.CountOffline(absInputSource, concurrency, customCfg != nil)
		if err != nil {
			util.Fatal("Error during discovery phase: %v", err)
		}
		resumeMgr.SaveTotal(total)
	}
	tracker.AddTotal(total)
	tracker.FinalizeTotal()

	offlineInputCh, err := input.ParseOffline(absInputSource, resumeMgr.IsCompleted, concurrency, customCfg)
	if err != nil {
		util.Fatal("Error initializing offline parsing: %v", err)
	}

	offlineWorkerInputCh := make(chan *model.OfflineInput, 2000) // Stable buffer for memory
	resultChWorker := make(chan []model.Detection, 5000)      // Increased cushion for slow disks
	var wg sync.WaitGroup

	numWorkers := concurrency
	if numWorkers <= 0 {
		numWorkers = 1
	}

	// 1. Start the Parser (Producer) in background
	go func() {
		defer close(offlineWorkerInputCh)
		for offInput := range offlineInputCh {
			offlineWorkerInputCh <- offInput
		}
	}()

	// 2. Start the Workers (Consumers)
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for offInput := range offlineWorkerInputCh {
				// Handle SKIPPED items (Resume)
				if offInput.Skipped {
					tracker.IncrementSuccess()
					model.OfflineInputPool.Put(offInput)
					continue
				}

				id := offInput.Path
				if id == "" { id = offInput.URL }
				if id == "" { id = offInput.Domain }

				if resumeMgr.IsCompleted(id) {
					tracker.IncrementSuccess()
					model.OfflineInputPool.Put(offInput)
					continue
				}

				// PARALLEL EXTRACTION for Custom Configs
				if customCfg != nil {
					if len(offInput.RawJSON) > 0 {
						custom.PopulateFromJSON(offInput.RawJSON, offInput, customCfg)
					} else if len(offInput.RawRegex) > 0 {
						custom.PopulateFromRegex(offInput.RawRegex, offInput, customCfg)
					}
				}

				// Select Detection Strategy
				currentInputType := model.InputTypeOffline
				if headersOnly {
					currentInputType = model.SourceHeadersOnly
				} else if bodyOnly {
					currentInputType = model.SourceBodyOnly
				}

				// HEAVY OPERATION: The Regex Engine
				detections, err := engine.Detect(offInput.Headers, offInput.Body, currentInputType)
				if err != nil {
					util.Warn("Failed: %s (%v)", id, err)
					tracker.IncrementError()
					model.OfflineInputPool.Put(offInput)
					continue
				}

				for i := range detections {
					detections[i].Domain = offInput.Domain
					detections[i].URL = offInput.URL
				}

				resultChWorker <- detections
				resumeMgr.MarkCompleted(id)
				tracker.IncrementSuccess()

				// RECYCLE
				model.OfflineInputPool.Put(offInput)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultChWorker)
	}()

	return tracker, resultChWorker
}

func runOnline(inputSource string, engine *detect.WappalyzerEngine) (*progress.Tracker, <-chan []model.Detection) {
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
				
				headers, body, err := online.FetchOnline(target, timeout)
				if err != nil {
					util.Warn("Failed: %s (%v)", target.URL, err)
					tracker.IncrementError()
					continue
				}

				detections, err := engine.Detect(headers, body, model.SourceWappalyzer)
				if err != nil {
					util.Warn("Failed to detect for %s: %v", target.URL, err)
					tracker.IncrementError()
					continue
				}

				for i := range detections {
					detections[i].Domain = target.Domain
					detections[i].URL = target.URL
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
	rootCmd.PersistentFlags().StringVar(&proxyAddr, "proxy", "", "Start a proxy server on this address (e.g., :8080) to passively scan traffic")
	rootCmd.PersistentFlags().StringVar(&inputConfigPath, "input-config", "", "YAML file defining custom input parsing (JSON or Regex support)")
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
	rootCmd.PersistentFlags().IntVarP(&concurrency, "concurrency", "c", runtime.NumCPU()*2, "Number of concurrent workers (goroutines)")
	rootCmd.PersistentFlags().IntVarP(&concurrency, "threads", "t", runtime.NumCPU()*2, "Number of concurrent workers (alias for --concurrency)")
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
	rootCmd.PersistentFlags().BoolVar(&showNuclei, "nuclei", false, "Generate Nuclei tags and recommended scan command")
}
