package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mikey/faultline/internal/config"
)

func TestEngineIngestsGenericLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	if err := os.WriteFile(logPath, []byte("ERROR TypeError: boom\n    at charge (/app/src/pay.js:81:1)\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Sources: []config.SourceConfig{{Name: "app", Type: "generic", Path: logPath}},
		Editor:  config.EditorConfig{Command: "code"},
	}
	eng := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := eng.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	defer eng.Stop()

	deadline := time.After(3 * time.Second)
	for {
		if eng.Store().Len() >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for event")
		case <-time.After(50 * time.Millisecond):
		}
	}
	items := eng.Store().List("")
	if items[0].Type != "TypeError" {
		t.Fatalf("type = %q", items[0].Type)
	}
}
