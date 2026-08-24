package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the project-level Faultline configuration.
type Config struct {
	Sources       []SourceConfig `yaml:"sources"`
	Notifications NotifyConfig   `yaml:"notifications"`
	Editor        EditorConfig   `yaml:"editor"`
}

// SourceConfig describes a single log source.
type SourceConfig struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"` // laravel | apache
	Path string `yaml:"path"`
}

// NotifyConfig controls desktop notifications.
type NotifyConfig struct {
	Enabled bool `yaml:"enabled"`
	Sound   bool `yaml:"sound"`
}

// EditorConfig controls opening files at a location.
type EditorConfig struct {
	// Command is one of: code, phpstorm, or a custom template containing
	// {file} and optionally {line}.
	Command string `yaml:"command"`
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
	}
}

// Validate checks required fields.
func (c *Config) Validate() error {
	if len(c.Sources) == 0 {
		return fmt.Errorf("config: at least one source is required")
	}
	for i, s := range c.Sources {
		if s.Path == "" {
			return fmt.Errorf("config: sources[%d].path is required", i)
		}
		switch s.Type {
		case "laravel", "apache":
		default:
			return fmt.Errorf("config: sources[%d].type must be laravel or apache, got %q", i, s.Type)
		}
	}
	return nil
}

// DefaultEnabledNotifications returns a notify config with notifications on.
func DefaultEnabledNotifications() NotifyConfig {
	return NotifyConfig{Enabled: true, Sound: false}
}
