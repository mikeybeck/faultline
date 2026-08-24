package statefile

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// State remembers the last opened project for the desktop app.
type State struct {
	ProjectDir string `yaml:"projectDir" json:"projectDir"`
	ConfigPath string `yaml:"configPath" json:"configPath"`
}

func dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "faultline"), nil
}

// Path is ~/.config/faultline/state.yaml (or OS equivalent).
func Path() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "state.yaml"), nil
}

func Load() (*State, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return nil, fmt.Errorf("read state: %w", err)
	}
	var st State
	if err := yaml.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return &st, nil
}

func Save(st State) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	data, err := yaml.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}
