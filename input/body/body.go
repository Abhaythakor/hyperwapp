package body

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Abhaythakor/hyperwapp/model"
	"github.com/Abhaythakor/hyperwapp/util"
)

// ParseBodyOnly parses a file or directory as raw body input.
func ParseBodyOnly(path string) (<-chan model.OfflineInput, error) {
	outputCh := make(chan model.OfflineInput)

	go func() {
		defer close(outputCh)

		fileInfo, err := os.Stat(path)
		if err != nil {
			util.Warn("Failed to stat path %s: %v", path, err)
			return
		}

		if fileInfo.IsDir() {
			err = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
				if err != nil {
					util.Warn("Error walking directory %s: %v", p, err)
					return nil
				}
				if !d.IsDir() {
					processSingleFile(p, outputCh)
				}
				return nil
			})
		} else {
			processSingleFile(path, outputCh)
		}
	}()

	return outputCh, nil
}

func processSingleFile(path string, outputCh chan<- model.OfflineInput) {
	body, err := os.ReadFile(path)
	if err != nil {
		util.Warn("Failed to read body-only file %s: %v", path, err)
		return
	}

	if len(body) == 0 {
		return
	}

	input := model.OfflineInput{
		Domain:  inferDomain(path),
		URL:     "",
		Headers: make(map[string][]string),
		Body:    body,
	}
	util.Debug("Created Body-Only OfflineInput for file: %s (Domain: %s)", path, input.Domain)
	outputCh <- input
}

// inferDomain attempts to infer a domain from a file path.
// This is a simple heuristic for body-only files.
func inferDomain(filePath string) string {
	fileName := filepath.Base(filePath)
	// Remove extension
	ext := filepath.Ext(fileName)
	if ext != "" {
		fileName = strings.TrimSuffix(fileName, ext)
	}
	// Basic check for common domain formats in filenames
	return fileName
}
