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
func ParseCustom(path string, cc *CompiledConfig, skipFunc func(string) bool, concurrency int) (<-chan *model.OfflineInput, error) {
	outputCh := make(chan *model.OfflineInput)

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

func processCustomFile(path string, outputCh chan<- *model.OfflineInput, cc *CompiledConfig, skipFunc func(string) bool) {
	file, err := os.Open(path)
	if err != nil {
		util.Warn("Failed to open file %s: %v", path, err)
		return
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, 1024*1024) // 1MB read buffer

	if cc.Config.Format == "json" {
		lineNum := 0
		for {
			lineNum++
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				uniqueID := fmt.Sprintf("%s#L%d", path, lineNum)
				
				// FAST SKIP: Check resume log before doing anything else
				if skipFunc != nil && skipFunc(uniqueID) {
					input := model.OfflineInputPool.Get().(*model.OfflineInput)
					input.Reset()
					input.Path = uniqueID
					input.Skipped = true
					outputCh <- input
					if err != nil { break }
					continue
				}

				// Move the JSON parsing work into the channel
				// We wrap the raw data in OfflineInput so the WORKERS handle the GJSON work.
				input := model.OfflineInputPool.Get().(*model.OfflineInput)
				input.Reset()
				input.Path = uniqueID
				input.RawJSON = make([]byte, len(line))
				copy(input.RawJSON, line) // Must copy because line is a reuse buffer
				
				outputCh <- input
			}
			if err != nil {
				break
			}
		}
	} else if cc.Config.Format == "regex" {
		// Similar optimization for line-based Regex
		if cc.Config.Regex.RecordSeparator == "\n" || cc.Config.Regex.RecordSeparator == "" {
			lineNum := 0
			for {
				lineNum++
				line, err := reader.ReadBytes('\n')
				if len(line) > 0 {
					uniqueID := fmt.Sprintf("%s#R%d", path, lineNum)
					if skipFunc != nil && skipFunc(uniqueID) {
						input := model.OfflineInputPool.Get().(*model.OfflineInput)
						input.Reset()
						input.Path = uniqueID
						input.Skipped = true
						outputCh <- input
						if err != nil { break }
						continue
					}
					
					input := model.OfflineInputPool.Get().(*model.OfflineInput)
					input.Reset()
					input.Path = uniqueID
					input.RawRegex = make([]byte, len(line))
					copy(input.RawRegex, line)

					outputCh <- input
				}
				if err != nil {
					break
				}
			}
		} else {
			// For complex separators, we stick to the existing logic but keep it safe
			data, _ := os.ReadFile(path)
			records := cc.RecordSep.Split(string(data), -1)
			for _, record := range records {
				if strings.TrimSpace(record) == "" { continue }
				input := ExtractFromRegex([]byte(record), cc)
				if input != nil { outputCh <- input }
			}
		}
	}
}

func ExtractFromJSON(data []byte, cc *CompiledConfig) *model.OfflineInput {
	cfg := cc.Config.JSON
	res := gjson.ParseBytes(data)
	if !res.IsObject() {
		return nil
	}

	out := model.OfflineInputPool.Get().(*model.OfflineInput)
	out.Reset()

	out.URL = res.Get(cfg.URLPath).String()
	out.Domain = res.Get(cfg.DomainPath).String()
	out.Body = []byte(res.Get(cfg.BodyPath).String())

	// Headers (optimized extraction)
	if cfg.HeadersPath != "" {
		res.Get(cfg.HeadersPath).ForEach(func(key, value gjson.Result) bool {
			k := key.String()
			if value.IsArray() {
				for _, item := range value.Array() {
					out.Headers[k] = append(out.Headers[k], item.String())
				}
			} else {
				out.Headers[k] = []string{value.String()}
			}
			return true // continue
		})
	}

	if out.Domain == "" && out.URL != "" {
		if u, err := url.Parse(out.URL); err == nil {
			out.Domain = u.Hostname()
		}
	}

	return out
}

func ExtractFromRegex(record []byte, cc *CompiledConfig) *model.OfflineInput {
	recordStr := string(record)
	out := model.OfflineInputPool.Get().(*model.OfflineInput)
	out.Reset()

	if cc.URLRegex != nil {
		m := cc.URLRegex.FindStringSubmatch(recordStr)
		if len(m) > 1 {
			out.URL = m[1]
		}
	}
	if cc.DomainRegex != nil {
		m := cc.DomainRegex.FindStringSubmatch(recordStr)
		if len(m) > 1 {
			out.Domain = m[1]
		}
	}
	if cc.BodyRegex != nil {
		m := cc.BodyRegex.FindStringSubmatch(recordStr)
		if len(m) > 1 {
			out.Body = []byte(m[1])
		}
	}
	if cc.HeadersRegex != nil {
		m := cc.HeadersRegex.FindStringSubmatch(recordStr)
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
		model.OfflineInputPool.Put(out)
		return nil
	}
	return out
}
