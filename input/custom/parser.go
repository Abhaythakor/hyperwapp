package custom

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Abhaythakor/hyperwapp/model"
	"github.com/Abhaythakor/hyperwapp/util"
	"github.com/tidwall/gjson"
)

// ParseCustom handles parsing based on the YAML configuration.
func ParseCustom(path string, cc *CompiledConfig, skipFunc func(string) bool, concurrency int) (<-chan model.OfflineInput, error) {
	outputCh := make(chan model.OfflineInput)

	go func() {
		defer close(outputCh)

		fileInfo, err := os.Stat(path)
		if err != nil {
			util.Warn("Failed to stat path %s: %v", path, err)
			return
		}

		if fileInfo.IsDir() {
			_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				processCustomFile(p, outputCh, cc, skipFunc)
				return nil
			})
		} else {
			processCustomFile(path, outputCh, cc, skipFunc)
		}
	}()

	return outputCh, nil
}

func processCustomFile(path string, outputCh chan<- model.OfflineInput, cc *CompiledConfig, skipFunc func(string) bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		util.Warn("Failed to read file %s: %v", path, err)
		return
	}

	if cc.Config.Format == "json" {
		// Handle JSONL (line by line) or Single JSON
		content := string(data)
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			uniqueID := fmt.Sprintf("%s#L%d", path, i+1)
			if skipFunc != nil && skipFunc(uniqueID) {
				outputCh <- model.OfflineInput{Path: uniqueID, Skipped: true}
				continue
			}
			input := extractFromJSON([]byte(line), cc)
			if input != nil {
				input.Path = uniqueID
				outputCh <- *input
			}
		}
	} else if cc.Config.Format == "regex" {
		// Split by record separator
		records := cc.RecordSep.Split(string(data), -1)
		for i, record := range records {
			if strings.TrimSpace(record) == "" {
				continue
			}
			uniqueID := fmt.Sprintf("%s#R%d", path, i+1)
			if skipFunc != nil && skipFunc(uniqueID) {
				outputCh <- model.OfflineInput{Path: uniqueID, Skipped: true}
				continue
			}
			input := extractFromRegex(record, cc)
			if input != nil {
				input.Path = uniqueID
				outputCh <- *input
			}
		}
	}
}

func extractFromJSON(data []byte, cc *CompiledConfig) *model.OfflineInput {
	cfg := cc.Config.JSON
	res := gjson.ParseBytes(data)
	if !res.IsObject() {
		return nil
	}

	out := &model.OfflineInput{Headers: make(map[string][]string)}
	out.URL = res.Get(cfg.URLPath).String()
	out.Domain = res.Get(cfg.DomainPath).String()
	out.Body = []byte(res.Get(cfg.BodyPath).String())

	// Headers
	hMap := res.Get(cfg.HeadersPath).Map()
	for k, v := range hMap {
		out.Headers[k] = []string{v.String()}
	}

	if out.Domain == "" && out.URL != "" {
		if u, err := url.Parse(out.URL); err == nil {
			out.Domain = u.Hostname()
		}
	}

	return out
}

func extractFromRegex(record string, cc *CompiledConfig) *model.OfflineInput {
	out := &model.OfflineInput{Headers: make(map[string][]string)}

	if cc.URLRegex != nil {
		m := cc.URLRegex.FindStringSubmatch(record)
		if len(m) > 1 {
			out.URL = m[1]
		}
	}
	if cc.DomainRegex != nil {
		m := cc.DomainRegex.FindStringSubmatch(record)
		if len(m) > 1 {
			out.Domain = m[1]
		}
	}
	if cc.BodyRegex != nil {
		m := cc.BodyRegex.FindStringSubmatch(record)
		if len(m) > 1 {
			out.Body = []byte(m[1])
		}
	}
	if cc.HeadersRegex != nil {
		m := cc.HeadersRegex.FindStringSubmatch(record)
		if len(m) > 1 {
			// Try parsing as JSON first, then as raw block
			var hMap map[string]interface{}
			if err := json.Unmarshal([]byte(m[1]), &hMap); err == nil {
				for k, v := range hMap {
					out.Headers[k] = []string{fmt.Sprint(v)}
				}
			} else {
				// Parse as standard HTTP header block
				scanner := bufio.NewScanner(strings.NewReader(m[1]))
				for scanner.Scan() {
					parts := strings.SplitN(scanner.Text(), ":", 2)
					if len(parts) == 2 {
						out.Headers[strings.TrimSpace(parts[0])] = append(out.Headers[strings.TrimSpace(parts[0])], strings.TrimSpace(parts[1]))
					}
				}
			}
		}
	}

	if out.Domain == "" && out.URL != "" {
		if u, err := url.Parse(out.URL); err == nil {
			out.Domain = u.Hostname()
		}
	}

	if out.Domain == "" && out.URL == "" {
		return nil
	}
	return out
}
