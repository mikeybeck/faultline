package engine

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mikey/faultline/internal/config"
	"github.com/mikey/faultline/internal/ingest"
	"github.com/mikey/faultline/internal/mark"
)

func testEngine() *Engine {
	e := New()
	e.SetIngestAddr(ingest.Disabled)
	return e
}

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
	eng := testEngine()
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

func TestEngineBackfillDoesNotNotify(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	f, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	const n = 20000
	const kinds = 10
	for i := 0; i < n; i++ {
		fmt.Fprintf(f, "ERROR TypeError: boom %d\n", i%kinds)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Sources: []config.SourceConfig{{Name: "app", Type: "generic", Path: logPath}},
		Editor:  config.EditorConfig{Command: "code"},
	}
	eng := testEngine()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := eng.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	defer eng.Stop()

	deadline := time.After(8 * time.Second)
	for {
		if eng.Store().Len() >= kinds {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for fingerprints, got %d", eng.Store().Len())
		case <-time.After(20 * time.Millisecond):
		}
	}
	// Let idle flush finish backfill.
	time.Sleep(900 * time.Millisecond)
	if got := eng.NotifyCalls(); got != 0 {
		t.Fatalf("notifications during backfill = %d, want 0", got)
	}

	fh, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fh.WriteString("ERROR TypeError: brand new live error\n"); err != nil {
		t.Fatal(err)
	}
	_ = fh.Close()

	deadline = time.After(3 * time.Second)
	for {
		if eng.Store().Len() >= kinds+1 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for live event, len=%d", eng.Store().Len())
		case <-time.After(20 * time.Millisecond):
		}
	}
	if got := eng.NotifyCalls(); got < 1 {
		t.Fatalf("expected a live notification, got %d", got)
	}
}

func TestEngineClearSkipsOnRestart(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	if err := os.WriteFile(logPath, []byte("ERROR TypeError: old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Sources: []config.SourceConfig{{Name: "app", Type: "generic", Path: logPath}},
		Editor:  config.EditorConfig{Command: "code"},
	}
	marks := mark.Open(filepath.Join(dir, "marks.yaml"))
	eng := testEngine()
	eng.UseMarks(marks)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := eng.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(3 * time.Second)
	for {
		if eng.Store().Len() >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for first ingest")
		case <-time.After(20 * time.Millisecond):
		}
	}
	time.Sleep(400 * time.Millisecond)
	eng.Clear()
	if !eng.HasMarks() {
		t.Fatal("expected a saved mark after clear")
	}
	eng.Stop()

	eng2 := testEngine()
	eng2.UseMarks(marks)
	if err := eng2.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	defer eng2.Stop()
	time.Sleep(600 * time.Millisecond)
	if n := eng2.Store().Len(); n != 0 {
		t.Fatalf("re-ingested marked content, len=%d", n)
	}

	fh, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fh.WriteString("ERROR TypeError: after-clear\n"); err != nil {
		t.Fatal(err)
	}
	_ = fh.Close()

	deadline = time.After(3 * time.Second)
	for {
		if eng2.Store().Len() >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for post-mark line")
		case <-time.After(20 * time.Millisecond):
		}
	}
	items := eng2.Store().List("")
	if items[0].Message != "TypeError: after-clear" && items[0].Type != "TypeError" {
		t.Fatalf("unexpected event: %+v", items[0])
	}
	if items[0].Message != "" && !contains(items[0].Message, "after-clear") && items[0].Raw != "" && !contains(items[0].Raw, "after-clear") && items[0].Type != "TypeError" {
		t.Fatalf("wanted after-clear, got %+v", items[0])
	}
}

func TestEngineClearMarksReingests(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	if err := os.WriteFile(logPath, []byte("ERROR TypeError: old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Sources: []config.SourceConfig{{Name: "app", Type: "generic", Path: logPath}},
		Editor:  config.EditorConfig{Command: "code"},
	}
	marks := mark.Open(filepath.Join(dir, "marks.yaml"))
	eng := testEngine()
	eng.UseMarks(marks)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := eng.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(3 * time.Second)
	for {
		if eng.Store().Len() >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for ingest")
		case <-time.After(20 * time.Millisecond):
		}
	}
	time.Sleep(400 * time.Millisecond)
	eng.Clear()
	eng.ClearMarks()
	if eng.HasMarks() {
		t.Fatal("mark should be gone")
	}
	eng.Stop()

	eng2 := testEngine()
	eng2.UseMarks(marks)
	if err := eng2.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	defer eng2.Stop()
	deadline = time.After(3 * time.Second)
	for {
		if eng2.Store().Len() >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for re-ingest after clearing mark")
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}

func TestEngineIngestsBrowserPOST(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	if err := os.WriteFile(logPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Sources: []config.SourceConfig{{Name: "app", Type: "generic", Path: logPath}},
		Editor:  config.EditorConfig{Command: "code"},
	}
	eng := New()
	eng.SetIngestAddr("127.0.0.1:0")
	eng.SetProjectDir(dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := eng.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	defer eng.Stop()

	var addr string
	deadline := time.After(2 * time.Second)
	for addr == "" {
		addr = eng.BoundIngest()
		if addr != "" {
			break
		}
		select {
		case <-deadline:
			t.Fatal("ingest did not bind")
		case <-time.After(10 * time.Millisecond):
		}
	}

	body := `{"type":"TypeError","message":"browser boom","file":"src/app.js","line":9,"stack":"TypeError: browser boom"}`
	resp, err := http.Post("http://"+addr+"/ingest", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status %d", resp.StatusCode)
	}

	deadline = time.After(2 * time.Second)
	for {
		if eng.Store().Len() >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for browser event")
		case <-time.After(20 * time.Millisecond):
		}
	}
	ev := eng.Store().List("")[0]
	if ev.Type != "TypeError" || ev.Source != "browser" {
		t.Fatalf("event = %+v", ev)
	}
	if ev.Message != "browser boom" {
		t.Fatalf("message = %q", ev.Message)
	}
}
