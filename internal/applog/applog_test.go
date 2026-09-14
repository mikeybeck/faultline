package applog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteCreatesLog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("LOCALAPPDATA", dir)

	if err := Write("settings failed", "goroutine 1\n"); err != nil {
		t.Fatal(err)
	}
	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(p) != filepath.Join(dir, "faultline") && !strings.Contains(p, "faultline") {
		t.Fatalf("path = %q", p)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "ERROR settings failed") {
		t.Fatalf("log = %q", got)
	}
	if !strings.Contains(got, "goroutine 1") {
		t.Fatalf("missing stack: %q", got)
	}
}
