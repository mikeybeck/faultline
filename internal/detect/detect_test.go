package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSniffLaravelAndApacheAndGeneric(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	if got := Sniff(filepath.Join(root, "laravel_sample.log")); got != TypeLaravel {
		t.Fatalf("laravel sniff = %q", got)
	}
	if got := Sniff(filepath.Join(root, "apache_sample.log")); got != TypeApache {
		t.Fatalf("apache sniff = %q", got)
	}
	if got := Sniff(filepath.Join(root, "generic_sample.log")); got != TypeGeneric {
		t.Fatalf("generic sniff = %q", got)
	}
}

func TestScanDirFindsLogsAndSkipsVendor(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "vendor", "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "storage", "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	appLog := filepath.Join(dir, "logs", "app.log")
	if err := os.WriteFile(appLog, []byte("ERROR boom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vendor", "logs", "ignore.log"), []byte("ERROR no\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	laravel := filepath.Join(dir, "storage", "logs", "laravel.log")
	if err := os.WriteFile(laravel, []byte("[2026-07-15 14:31:01] local.ERROR: TypeError: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("candidates = %d, want 2: %+v", len(got), got)
	}
	types := map[string]string{}
	for _, c := range got {
		types[filepath.Base(c.Path)] = c.Type
	}
	if types["app.log"] != TypeGeneric {
		t.Errorf("app.log type = %q", types["app.log"])
	}
	if types["laravel.log"] != TypeLaravel {
		t.Errorf("laravel.log type = %q", types["laravel.log"])
	}
}
