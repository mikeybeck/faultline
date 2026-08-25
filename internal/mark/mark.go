package mark

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/mikey/faultline/internal/source"
)

// Store persists resume offsets so a later launch can skip already-cleared log content.
type Store struct {
	mu   sync.Mutex
	path string
}

type fileDoc struct {
	Files map[string]source.Resume `yaml:"files"`
}

// Default is ~/.config/faultline/marks.yaml (or the OS equivalent).
func Default() *Store {
	base, err := os.UserConfigDir()
	if err != nil {
		return Open("")
	}
	return Open(filepath.Join(base, "faultline", "marks.yaml"))
}

// Open returns a store rooted at path. An empty path is a no-op store.
func Open(path string) *Store {
	return &Store{path: path}
}

func key(path string) string {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func (s *Store) load() fileDoc {
	doc := fileDoc{Files: map[string]source.Resume{}}
	if s == nil || s.path == "" {
		return doc
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return doc
	}
	if err := yaml.Unmarshal(data, &doc); err != nil || doc.Files == nil {
		doc.Files = map[string]source.Resume{}
	}
	return doc
}

func (s *Store) save(doc fileDoc) error {
	if s == nil || s.path == "" {
		return nil
	}
	if doc.Files == nil {
		doc.Files = map[string]source.Resume{}
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("save marks: %w", err)
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("save marks: %w", err)
	}
	return os.WriteFile(s.path, data, 0o644)
}

// Get returns the resume point for a log path, if any.
func (s *Store) Get(logPath string) (source.Resume, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(logPath)
	if k == "" {
		return source.Resume{}, false
	}
	r, ok := s.load().Files[k]
	if !ok || r.Identity == 0 {
		return source.Resume{}, false
	}
	return r, true
}

// PutMany merges resume points for the given log paths.
func (s *Store) PutMany(marks map[string]source.Resume) error {
	if s == nil || len(marks) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc := s.load()
	for path, r := range marks {
		k := key(path)
		if k == "" || r.Identity == 0 {
			continue
		}
		doc.Files[k] = r
	}
	return s.save(doc)
}

// Delete removes resume points for the given log paths.
func (s *Store) Delete(logPaths ...string) error {
	if s == nil || len(logPaths) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc := s.load()
	changed := false
	for _, path := range logPaths {
		k := key(path)
		if k == "" {
			continue
		}
		if _, ok := doc.Files[k]; ok {
			delete(doc.Files, k)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.save(doc)
}

// HasAny reports whether any of the log paths has a saved mark.
func (s *Store) HasAny(logPaths ...string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc := s.load()
	for _, path := range logPaths {
		k := key(path)
		if k == "" {
			continue
		}
		if r, ok := doc.Files[k]; ok && r.Identity != 0 {
			return true
		}
	}
	return false
}
