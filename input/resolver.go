package input

import (
	"bufio"
	"fmt" // Added fmt package
	"io"
	"os"
	"strings"

	"hyperwapp/model"
	"hyperwapp/util"
)

// ResolveInput takes an input source (file, stdin, or direct arg) and returns a slice of targets.
func ResolveInput(input string, offlineMode bool) ([]model.Target, error) {
	var targets []model.Target

	if input == "-" { // Read from stdin
		util.Debug("Reading input from stdin")
		return readInputFromReader(os.Stdin, offlineMode)
	}

	fileInfo, err := os.Stat(input)
	if err == nil && !fileInfo.IsDir() { // Input is a regular file
		util.Debug("Reading input from file: %s", input)
		file, err := os.Open(input)
		if err != nil {
			return nil, fmt.Errorf("failed to open input file %s: %w", input, err)
		}
		defer file.Close()
		return readInputFromReader(file, offlineMode)
	} else if input != "" { // Direct input (URL or path for offline)
		// If it's a directory and we aren't in offline mode, this is an error for online mode
		if err == nil && fileInfo.IsDir() && !offlineMode {
			return nil, fmt.Errorf("input '%s' is a directory, but -offline flag was not set", input)
		}

		util.Debug("Processing direct input: %s", input)
		target, err := normalizeTarget(input, offlineMode)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	return targets, nil
}

func readInputFromReader(reader io.Reader, offlineMode bool) ([]model.Target, error) {
	var targets []model.Target
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		target, err := normalizeTarget(line, offlineMode)
		if err != nil {
			util.Warn("Skipping invalid input line '%s': %v", line, err)
			continue
		}
		targets = append(targets, target)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}
	return targets, nil
}
