package custom

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Config defines how to extract data from custom input formats.
type Config struct {
	// General
	Format string `yaml:"format"` // "json" or "regex"

	// JSON specific (GJSON paths)
	JSON struct {
		URLPath     string `yaml:"url_path"`
		DomainPath  string `yaml:"domain_path"`
		HeadersPath string `yaml:"headers_path"`
		BodyPath    string `yaml:"body_path"`
	} `yaml:"json"`

	// Regex specific
	Regex struct {
		RecordSeparator string `yaml:"record_separator"` // Regex to split file into records
		URLRegex        string `yaml:"url_regex"`
		DomainRegex     string `yaml:"domain_regex"`
		HeadersRegex    string `yaml:"headers_regex"`    // Extract a block or JSON string
		BodyRegex       string `yaml:"body_regex"`
	} `yaml:"regex"`
}

// CompiledConfig holds the compiled regexes for performance
type CompiledConfig struct {
	Config          *Config
	RecordSep       *regexp.Regexp
	URLRegex        *regexp.Regexp
	DomainRegex     *regexp.Regexp
	HeadersRegex    *regexp.Regexp
	BodyRegex       *regexp.Regexp
}

// LoadConfig loads the custom parsing configuration from a YAML file.
func LoadConfig(path string) (*CompiledConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read input config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse input config YAML: %w", err)
	}

	cc := &CompiledConfig{Config: &cfg}

	// Compile regexes
	if cfg.Format == "regex" {
		if cfg.Regex.RecordSeparator != "" {
			cc.RecordSep = regexp.MustCompile(cfg.Regex.RecordSeparator)
		} else {
			cc.RecordSep = regexp.MustCompile("\n") // Default line-by-line
		}
		if cfg.Regex.URLRegex != "" {
			cc.URLRegex = regexp.MustCompile(cfg.Regex.URLRegex)
		}
		if cfg.Regex.DomainRegex != "" {
			cc.DomainRegex = regexp.MustCompile(cfg.Regex.DomainRegex)
		}
		if cfg.Regex.HeadersRegex != "" {
			cc.HeadersRegex = regexp.MustCompile(cfg.Regex.HeadersRegex)
		}
		if cfg.Regex.BodyRegex != "" {
			cc.BodyRegex = regexp.MustCompile(cfg.Regex.BodyRegex)
		}
	}

	return cc, nil
}
