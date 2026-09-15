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

func TestValidateAcceptsCommandParser(t *testing.T) {
	if err := (&Config{Sources: []SourceConfig{{Type: "command", Path: "npm run dev", Parser: "json"}}}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (&Config{Sources: []SourceConfig{{Type: "command", Path: "npm run dev", Parser: "nope"}}}).Validate(); err == nil {
		t.Fatal("expected parser validation error")
	}
}

func TestValidateAcceptsCommandAndJSON(t *testing.T) {
	if err := (&Config{Sources: []SourceConfig{{Type: "command", Path: "npm run dev"}}}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (&Config{Sources: []SourceConfig{{Type: "json", Path: "app.log"}}}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAcceptsBrowserWithoutPath(t *testing.T) {
	cfg := &Config{Sources: []SourceConfig{{Type: "browser"}}}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.applyDefaults()
	if cfg.Sources[0].Path != "127.0.0.1:9477" {
		t.Fatalf("path = %q", cfg.Sources[0].Path)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "faultline.yaml")
	cfg := &Config{
		Sources: []SourceConfig{{Name: "app", Type: "generic", Path: "logs/app.log"}},
		Editor:  EditorConfig{Command: "code"},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Sources) != 1 || got.Sources[0].Type != "generic" {
		t.Fatalf("round trip = %+v", got.Sources)
	}
}

func TestSaveRoundTripFollowLatest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "faultline.yaml")
	cfg := &Config{
		Sources: []SourceConfig{{Name: "app", Type: "generic", Path: "logs/app.log"}},
		Inbox:   InboxConfig{FollowLatest: true},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Inbox.FollowLatest {
		t.Fatal("followLatest was not saved")
	}
}
