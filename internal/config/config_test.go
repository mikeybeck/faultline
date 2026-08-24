package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "faultline.yaml")
	content := `
sources:
  - name: laravel
    type: laravel
    path: storage/logs/laravel.log
  - name: apache
    type: apache
    path: /var/log/apache2/error.log
notifications:
  enabled: true
editor:
  command: phpstorm
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Sources) != 2 {
		t.Fatalf("sources = %d", len(cfg.Sources))
	}
	if cfg.Editor.Command != "phpstorm" {
		t.Fatalf("editor = %q", cfg.Editor.Command)
	}
}

func TestFind(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "faultline.yml"), []byte("sources:\n  - type: laravel\n    path: a.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	found, err := Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(found) != "faultline.yml" {
		t.Fatalf("found %q", found)
	}
}

func TestValidateRejectsBadType(t *testing.T) {
	cfg := &Config{Sources: []SourceConfig{{Type: "docker", Path: "x"}}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
