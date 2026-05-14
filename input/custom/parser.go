package custom

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Abhaythakor/hyperwapp/model"
	"github.com/Abhaythakor/hyperwapp/util"
	"github.com/tidwall/gjson"
)

// ParseCustom handles parsing based on the YAML configuration.
func ParseCustom(ctx context.Context, path string, cc *CompiledConfig, skipFunc func(string) bool, concurrency int) (<-chan *model.OfflineInput, error) {
	outputCh := make(chan *model.OfflineInput, 1000)

	go func() {
		defer close(outputCh)

		fileInfo, err := os.Stat(path)
		if err != nil {
			util.Warn("Failed to stat path %s: %v", path, err)
			return
		}

		if fileInfo.IsDir() {
			_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
				select {
				case <-ctx.Done():
					return filepath.SkipAll
				default:
				}

				if err != nil || d.IsDir() {
					return nil
				}
				processCustomFile(ctx, p, outputCh, cc, skipFunc, concurrency)
				return nil
			})
		} else {
			processCustomFile(ctx, path, outputCh, cc, skipFunc, concurrency)
		}
	}()

	return outputCh, nil
}

// processCustomFile uses a high-speed block reader to parallelize the "JSON Tax" processing.
func processCustomFile(ctx context.Context, path string, outputCh chan<- *model.OfflineInput, cc *CompiledConfig, skipFunc func(string) bool, concurrency int) {
	file, err := os.Open(path)
	if err != nil {
		util.Warn("Failed to open file %s: %v", path, err)
		return
	}
	defer file.Close()

	if cc.Config.Format != "json" {
		// Fallback for non-JSON formats (rarely used for 17GB files)
		processCustomFileLegacy(ctx, file, path, outputCh, cc, skipFunc)
		return
	}

	// HIGH SPEED JSONL PIPELINE
	// 1. Single sequential reader (Best for HDD)
	// 2. Worker pool for JSON decoding (Best for 2-core CPU)
	
	lineQueue := make(chan struct {
		lineNum int
		data    []byte
	}, 1000)

	var wg sync.WaitGroup
	// Use exactly 'concurrency' workers for parsing
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range lineQueue {
				uniqueID := fmt.Sprintf("%s#L%d", path, item.lineNum)
				
				// FAST SKIP: Check resume log
				if skipFunc != nil && skipFunc(uniqueID) {
					input := model.OfflineInputPool.Get().(*model.OfflineInput)
					input.Reset()
					input.Path = uniqueID
					input.Skipped = true
					outputCh <- input
					model.LinePool.Put(item.data)
					continue
				}

				input := model.OfflineInputPool.Get().(*model.OfflineInput)
				input.Reset()
				input.Path = uniqueID
				input.RawJSON = item.data // Direct assignment, no copy!

				select {
				case <-ctx.Done():
					return
				case outputCh <- input:
				}
			}
		}()
	}

	reader := bufio.NewReaderSize(file, 2*1024*1024) // 2MB read buffer
	lineNum := 0
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			lineNum++
			// Get a pooled buffer for the next line
			buf := model.LinePool.Get().([]byte)
			buf = append(buf[:0], line...)
			
			select {
			case <-ctx.Done():
				goto cleanup
			case lineQueue <- struct {
				lineNum int
				data    []byte
			}{lineNum, buf}:
			}
		}
		if err != nil {
			break
		}
	}

cleanup:
	close(lineQueue)
	wg.Wait()
}

// processCustomFileLegacy handles regex and other non-standard formats.
func processCustomFileLegacy(ctx context.Context, file *os.File, path string, outputCh chan<- *model.OfflineInput, cc *CompiledConfig, skipFunc func(string) bool) {
	reader := bufio.NewReaderSize(file, 1024*1024)
	lineNum := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
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
					continue
				}
				
				buf := model.LinePool.Get().([]byte)
				buf = append(buf[:0], line...)

				input := model.OfflineInputPool.Get().(*model.OfflineInput)
				input.Reset()
				input.Path = uniqueID
				input.RawRegex = buf

				select {
				case <-ctx.Done():
					return
				case outputCh <- input:
				}
			}
			if err != nil {
				return
			}
		}
	}
}

func PopulateFromJSON(data []byte, out *model.OfflineInput, cc *CompiledConfig) {
	cfg := cc.Config.JSON
	
	// Optimization: Use GetManyBytes for faster multi-path extraction in a single pass
	results := gjson.GetManyBytes(data, cfg.URLPath, cfg.DomainPath, cfg.BodyPath)
	
	out.URL = results[0].String()
	out.Domain = results[1].String()
	out.Body = []byte(results[2].String())

	// Headers (populate existing map)
	if cfg.HeadersPath != "" {
		gjson.GetBytes(data, cfg.HeadersPath).ForEach(func(key, value gjson.Result) bool {
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
}

func PopulateFromRegex(record []byte, out *model.OfflineInput, cc *CompiledConfig) {
	recordStr := string(record)

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
}
