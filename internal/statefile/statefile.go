package statefile

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const recentCap = 8

// Project is a remembered project folder.
type Project struct {
	Dir        string `yaml:"dir" json:"dir"`
	ConfigPath string `yaml:"configPath" json:"configPath"`
}

// State remembers the last opened project for the desktop app.
type State struct {
	ProjectDir string    `yaml:"projectDir" json:"projectDir"`
	ConfigPath string    `yaml:"configPath" json:"configPath"`
	Recent     []Project `yaml:"recent,omitempty" json:"recent"`
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

// Remember records the current project and prepends it to the recent list.
func Remember(projectDir, configPath string) error {
	st, err := Load()
	if err != nil {
		st = &State{}
	}
	st.ProjectDir = projectDir
	st.ConfigPath = configPath
	st.Recent = prependRecent(st.Recent, Project{Dir: projectDir, ConfigPath: configPath})
	return Save(*st)
}

// RecentProjects returns remembered projects whose config still exists.
func RecentProjects() []Project {
	st, err := Load()
	if err != nil {
		return nil
	}
	out := make([]Project, 0, len(st.Recent))
	seen := map[string]bool{}
	for _, p := range st.Recent {
		if p.ConfigPath == "" || seen[p.ConfigPath] {
			continue
		}
		if _, err := os.Stat(p.ConfigPath); err != nil {
			continue
		}
		seen[p.ConfigPath] = true
		out = append(out, p)
	}
	return out
}

func prependRecent(list []Project, p Project) []Project {
	if p.ConfigPath == "" {
		return list
	}
	out := []Project{p}
	for _, prev := range list {
		if prev.ConfigPath == p.ConfigPath {
			continue
		}
		out = append(out, prev)
		if len(out) >= recentCap {
			break
		}
	}
	return out
}
