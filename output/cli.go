package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings" // Added strings

	"hyperwapp/aggregate"
	"hyperwapp/model"
	"hyperwapp/util"
)

// CLIWriter implements the Writer interface for CLI output.
type CLIWriter struct {
	color    *util.Colorizer
	mode     string
	tempFile *os.File
}

// NewCLIWriter creates a new CLIWriter.
func NewCLIWriter(colorize bool) *CLIWriter {
	return &CLIWriter{
		color: util.NewColorizer(colorize),
		mode:  "all",
	}
}

// Write outputs detections for individual targets to the console or buffers them for domain mode.
func (w *CLIWriter) Write(detections []model.Detection) error {
	if len(detections) == 0 {
		return nil
	}

	if w.mode == "domain" {
		if w.tempFile == nil {
			var err error
			w.tempFile, err = os.CreateTemp("", "HyperWapp-cli-*.jsonl")
			if err != nil {
				return err
			}
		}
		encoder := json.NewEncoder(w.tempFile)
		for _, d := range detections {
			if err := encoder.Encode(d); err != nil {
				return err
			}
		}
		return nil
	}

	// Default "all" mode: compact single-line format
	targets := make(map[string][]string)
	for _, d := range detections {
		key := d.URL
		if key == "" {
			key = d.Domain
		}
		targets[key] = append(targets[key], w.color.Green(d.Technology))
	}

	for target, techs := range targets {
		if len(techs) == 0 {
			continue
		}
		// news.airbnb.com [jQuery CDN, MySQL, ...]
		fmt.Fprintf(os.Stdout, "%s [%s]\n", w.color.Cyan(target), strings.Join(techs, ", "))
	}

	return nil
}

// SetMode updates the output mode.
func (w *CLIWriter) SetMode(mode string) {
	w.mode = mode
}

// WriteAggregated is a manual override if needed.
func (w *CLIWriter) WriteAggregated(aggregated []aggregate.AggregatedDomain) error {
	for _, agg := range aggregated {
		if len(agg.Detections) == 0 {
			continue
		}

		fmt.Fprintf(os.Stdout, "\nDomain: %s\n", w.color.Cyan(agg.Domain))
		fmt.Fprintf(os.Stdout, "  URLs Scanned: %d\n", len(agg.URLs))
		// For very large scans, we might want to skip printing 10M URLs here
		if len(agg.URLs) < 50 {
			for _, u := range agg.URLs {
				fmt.Fprintf(os.Stdout, "    - %s\n", u)
			}
		} else {
			fmt.Fprintf(os.Stdout, "    - (and %d more URLs...)\n", len(agg.URLs)-1)
		}
		fmt.Fprintln(os.Stdout)

		fmt.Fprintf(os.Stdout, "  Technologies:\n")
		uniqueTechs := make(map[string]struct{})
		for _, d := range agg.Detections {
			uniqueTechs[d.Technology] = struct{}{}
		}
		var sortedTechs []string
		for tech := range uniqueTechs {
			sortedTechs = append(sortedTechs, tech)
		}
		sort.Strings(sortedTechs)

		for _, tech := range sortedTechs {
			fmt.Fprintf(os.Stdout, "    - %s\n", w.color.Green(tech))
		}
		fmt.Fprintln(os.Stdout)
	}
	return nil
}

// Close finalizes output, performing aggregation if in domain mode.
func (w *CLIWriter) Close() {
	if w.mode == "domain" && w.tempFile != nil {
		defer os.Remove(w.tempFile.Name())
		defer w.tempFile.Close()

		domainMap := make(map[string]*aggregate.AggregatedDomain)
		w.tempFile.Seek(0, 0)
		decoder := json.NewDecoder(w.tempFile)
		for {
			var d model.Detection
			if err := decoder.Decode(&d); err != nil {
				break
			}
			if _, ok := domainMap[d.Domain]; !ok {
				domainMap[d.Domain] = &aggregate.AggregatedDomain{
					Domain: d.Domain,
				}
			}
			found := false
			for _, u := range domainMap[d.Domain].URLs {
				if u == d.URL {
					found = true
					break
				}
			}
			if !found && d.URL != "" {
				domainMap[d.Domain].URLs = append(domainMap[d.Domain].URLs, d.URL)
			}
			domainMap[d.Domain].Detections = append(domainMap[d.Domain].Detections, d)
		}

		var sortedDomains []string
		for d := range domainMap {
			sortedDomains = append(sortedDomains, d)
		}
		sort.Strings(sortedDomains)

		for _, d := range sortedDomains {
			w.WriteAggregated([]aggregate.AggregatedDomain{*domainMap[d]})
		}
	}
}
