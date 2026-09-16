package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the project-level Faultline configuration.
type Config struct {
	Sources       []SourceConfig `yaml:"sources" json:"sources"`
	Notifications NotifyConfig   `yaml:"notifications" json:"notifications"`
	Editor        EditorConfig   `yaml:"editor" json:"editor"`
	Inbox         InboxConfig    `yaml:"inbox,omitempty" json:"inbox"`
	Browser       BrowserConfig  `yaml:"browser,omitempty" json:"browser"`
}

// SourceConfig describes a single log source.
type SourceConfig struct {
	Name   string `yaml:"name" json:"name"`
	Type   string `yaml:"type" json:"type"`               // generic | laravel | apache | json | browser | command
	Path   string `yaml:"path" json:"path"`               // file path, ingest addr, or shell command
	Parser string `yaml:"parser,omitempty" json:"parser"` // for command sources: generic | json | laravel | apache
}

// BrowserConfig is project-wide browser ingest extras.
type BrowserConfig struct {
	ExtraHosts []string `yaml:"extraHosts,omitempty" json:"extraHosts"`
}

// NotifyConfig controls desktop notifications.
type NotifyConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	Sound   bool `yaml:"sound" json:"sound"`
}

// InboxConfig controls desktop inbox behavior.
type InboxConfig struct {
	FollowLatest  bool         `yaml:"followLatest" json:"followLatest"`
	ClearOnCommit bool         `yaml:"clearOnCommit" json:"clearOnCommit"`
	Ignore        []IgnoreRule `yaml:"ignore,omitempty" json:"ignore,omitempty"`
	Views         []SavedView  `yaml:"views,omitempty" json:"views,omitempty"`
}

// SavedView is a named inbox filter.
type SavedView struct {
	Name     string `yaml:"name" json:"name"`
	Filter   string `yaml:"filter,omitempty" json:"filter,omitempty"`
	Severity string `yaml:"severity,omitempty" json:"severity,omitempty"`
	Source   string `yaml:"source,omitempty" json:"source,omitempty"`
	Sort     string `yaml:"sort,omitempty" json:"sort,omitempty"`
}

// EditorConfig controls opening files at a location.
type EditorConfig struct {
	// Command is one of: code, cursor, phpstorm; a full path to an
	// executable; or a custom template containing {file} and optionally {line}.
	Command string `yaml:"command" json:"command"`
}

// Load reads and validates a YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.applyDefaults()
	return &cfg, nil
}

// Find searches common config filenames in dir (or cwd).
func Find(dir string) (string, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	candidates := []string{
		"faultline.yaml",
		"faultline.yml",
		".faultline.yaml",
		".faultline.yml",
	}
	for _, name := range candidates {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("no faultline.yaml found in %s", dir)
}

func (c *Config) applyDefaults() {
	if c.Editor.Command == "" {
		c.Editor.Command = "code"
	}
	if !c.Notifications.Enabled && c.Notifications == (NotifyConfig{}) {
		// Leave zero values alone; explicit default once fields present.
	}
	for i := range c.Sources {
		if c.Sources[i].Name == "" {
			c.Sources[i].Name = fmt.Sprintf("%s-%d", c.Sources[i].Type, i+1)
		}
		if c.Sources[i].Type == "" {
			c.Sources[i].Type = "generic"
		}
		if c.Sources[i].Type == "browser" && c.Sources[i].Path == "" {
			c.Sources[i].Path = "127.0.0.1:9477"
		}
	}
}

// Validate checks required fields.
func (c *Config) Validate() error {
	if len(c.Sources) == 0 {
		return fmt.Errorf("config: at least one source is required")
	}
	for i, s := range c.Sources {
		switch s.Type {
		case "browser":
			continue
		case "generic", "laravel", "apache", "json", "command":
		default:
			return fmt.Errorf("config: sources[%d].type must be generic, laravel, apache, json, browser, or command, got %q", i, s.Type)
		}
		if s.Path == "" {
			return fmt.Errorf("config: sources[%d].path is required", i)
		}
		if s.Type == "command" && s.Parser != "" {
			switch s.Parser {
			case "generic", "laravel", "apache", "json":
			default:
				return fmt.Errorf("config: sources[%d].parser must be generic, laravel, apache, or json, got %q", i, s.Parser)
			}
		}
	}
	if err := validateIgnore(c.Inbox.Ignore); err != nil {
		return err
	}
	for i, v := range c.Inbox.Views {
		if strings.TrimSpace(v.Name) == "" {
			return fmt.Errorf("config: inbox.views[%d].name is required", i)
		}
	}
	return nil
}

// IsFileSource reports whether typ tails a log file (not browser or command).
func IsFileSource(typ string) bool {
	switch typ {
	case "generic", "laravel", "apache", "json", "":
		return true
	default:
		return false
	}
}

// DefaultEnabledNotifications returns a notify config with notifications on.
func DefaultEnabledNotifications() NotifyConfig {
	return NotifyConfig{Enabled: true, Sound: false}
}

// Save writes cfg as YAML to path.
func Save(path string, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("save config: nil config")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	cfg.applyDefaults()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}
