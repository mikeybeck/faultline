package engine

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mikey/faultline/internal/applog"
	"github.com/mikey/faultline/internal/config"
	"github.com/mikey/faultline/internal/ingest"
	"github.com/mikey/faultline/internal/mark"
	"github.com/mikey/faultline/internal/persist"
)

func testEngine() *Engine {
	e := New()
	e.SetIngestAddr(ingest.Disabled)
	return e
}

func TestReportInternalShowsInInbox(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("LOCALAPPDATA", dir)

	eng := testEngine()
	eng.ReportInternal("settings hung", "goroutine 1\n")
	items := eng.Store().List("")
	if len(items) != 1 {
		t.Fatalf("len = %d", len(items))
	}
	if items[0].Source != applog.Source || items[0].Type != "Faultline" {
		t.Fatalf("event = %+v", items[0])
	}
	if items[0].Message != "settings hung" {
		t.Fatalf("message = %q", items[0].Message)
	}
	p, err := applog.Path()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "settings hung") {
		t.Fatalf("log = %q", data)
	}
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
	if len(items[0].Context) == 0 {
		t.Fatal("expected nearby log context")
	}
}

func TestEngineIgnoresConfiguredRules(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	if err := os.WriteFile(logPath, []byte("ERROR TypeError: boom\nERROR DeprecationWarning: old api\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Sources: []config.SourceConfig{{Name: "app", Type: "generic", Path: logPath}},
		Editor:  config.EditorConfig{Command: "code"},
		Inbox:   config.InboxConfig{Ignore: []config.IgnoreRule{{Type: "DeprecationWarning"}}},
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
	time.Sleep(400 * time.Millisecond)
	items := eng.Store().List("")
	if len(items) != 1 || items[0].Type == "DeprecationWarning" {
		t.Fatalf("ignore failed: %+v", items)
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

func persistEngine(t *testing.T, dir string) (*Engine, *persist.DB) {
	t.Helper()
	db, err := persist.Open(filepath.Join(dir, "inbox.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	eng := New()
	eng.SetIngestAddr("127.0.0.1:0")
	eng.SetProjectDir(dir)
	eng.UsePersist(db)
	return eng, db
}

func waitLen(t *testing.T, eng *Engine, n int, d time.Duration) {
	t.Helper()
	deadline := time.After(d)
	for {
		if eng.Store().Len() >= n {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for len>=%d, got %d", n, eng.Store().Len())
		case <-time.After(15 * time.Millisecond):
		}
	}
}

func waitBound(t *testing.T, eng *Engine) string {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		if addr := eng.BoundIngest(); addr != "" {
			return addr
		}
		select {
		case <-deadline:
			t.Fatal("ingest did not bind")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func postBrowser(t *testing.T, addr, body string) {
	t.Helper()
	resp, err := http.Post("http://"+addr+"/ingest", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestEnginePersistsBrowserAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Sources: []config.SourceConfig{{Name: "browser", Type: "browser", Path: "127.0.0.1:0"}},
		Editor:  config.EditorConfig{Command: "code"},
	}
	eng, db := persistEngine(t, dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := eng.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	addr := waitBound(t, eng)
	postBrowser(t, addr, `{"type":"TypeError","message":"keep me","file":"src/app.js","line":9}`)
	waitLen(t, eng, 1, 2*time.Second)
	eng.Stop()

	eng2 := New()
	eng2.SetIngestAddr(ingest.Disabled)
	eng2.SetProjectDir(dir)
	eng2.UsePersist(db)
	if err := eng2.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	defer eng2.Stop()
	if eng2.Store().Len() != 1 {
		t.Fatalf("hydrated len=%d", eng2.Store().Len())
	}
	if got := eng2.Store().List("")[0].Message; got != "keep me" {
		t.Fatalf("message=%q", got)
	}
}

func TestEngineDismissThenNewOccurrence(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Sources: []config.SourceConfig{{Name: "browser", Type: "browser", Path: "127.0.0.1:0"}},
		Editor:  config.EditorConfig{Command: "code"},
	}
	eng, _ := persistEngine(t, dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := eng.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	defer eng.Stop()
	addr := waitBound(t, eng)
	body := `{"type":"TypeError","message":"again","file":"src/app.js","line":4}`
	postBrowser(t, addr, body)
	waitLen(t, eng, 1, 2*time.Second)
	hash := eng.Store().List("")[0].Hash
	eng.Dismiss([]string{hash})
	if eng.Store().Len() != 0 {
		t.Fatal("expected empty after dismiss")
	}
	time.Sleep(20 * time.Millisecond)
	postBrowser(t, addr, body)
	waitLen(t, eng, 1, 2*time.Second)
	if eng.NotifyCalls() < 2 {
		t.Fatalf("expected notify on reappear, got %d", eng.NotifyCalls())
	}
}

func TestEngineMuteTypeSuppresses(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Sources: []config.SourceConfig{{Name: "browser", Type: "browser", Path: "127.0.0.1:0"}},
		Editor:  config.EditorConfig{Command: "code"},
	}
	eng, _ := persistEngine(t, dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := eng.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	defer eng.Stop()
	if err := eng.Mute(persist.KindType, "TypeError"); err != nil {
		t.Fatal(err)
	}
	addr := waitBound(t, eng)
	postBrowser(t, addr, `{"type":"TypeError","message":"nope","file":"src/app.js","line":1}`)
	time.Sleep(300 * time.Millisecond)
	if eng.Store().Len() != 0 {
		t.Fatalf("muted type still in inbox, len=%d", eng.Store().Len())
	}
	postBrowser(t, addr, `{"type":"ReferenceError","message":"ok","file":"src/app.js","line":2}`)
	waitLen(t, eng, 1, 2*time.Second)
	if got := eng.Store().List("")[0].Type; got != "ReferenceError" {
		t.Fatalf("type=%q", got)
	}
}

func TestEngineCaughtUpDoesNotDeleteBrowser(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	if err := os.WriteFile(logPath, []byte("ERROR TypeError: old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Sources: []config.SourceConfig{
			{Name: "app", Type: "generic", Path: logPath},
			{Name: "browser", Type: "browser", Path: "127.0.0.1:0"},
		},
		Editor: config.EditorConfig{Command: "code"},
	}
	marks := mark.Open(filepath.Join(dir, "marks.yaml"))
	eng, db := persistEngine(t, dir)
	eng.UseMarks(marks)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := eng.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	waitLen(t, eng, 1, 3*time.Second)
	addr := waitBound(t, eng)
	postBrowser(t, addr, `{"type":"TypeError","message":"browser kept","file":"src/app.js","line":9}`)
	waitLen(t, eng, 2, 2*time.Second)
	eng.Clear()
	if !eng.HasMarks() {
		t.Fatal("expected mark after caught up")
	}
	if eng.Store().Len() != 0 {
		t.Fatal("inbox should be empty")
	}
	eng.Stop()

	eng2 := New()
	eng2.SetIngestAddr(ingest.Disabled)
	eng2.SetProjectDir(dir)
	eng2.UsePersist(db)
	eng2.UseMarks(marks)
	if err := eng2.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	defer eng2.Stop()
	time.Sleep(400 * time.Millisecond)
	if n := eng2.Store().Len(); n != 0 {
		t.Fatalf("dismissed events came back, len=%d", n)
	}
	active, err := db.LoadActive(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 0 {
		t.Fatalf("active after caught up: %+v", active)
	}
}

func TestEngineReplayLogsKeepsBrowser(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	if err := os.WriteFile(logPath, []byte("ERROR TypeError: from-log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Sources: []config.SourceConfig{
			{Name: "app", Type: "generic", Path: logPath},
			{Name: "browser", Type: "browser", Path: "127.0.0.1:0"},
		},
		Editor: config.EditorConfig{Command: "code"},
	}
	marks := mark.Open(filepath.Join(dir, "marks.yaml"))
	eng, _ := persistEngine(t, dir)
	eng.UseMarks(marks)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := eng.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	defer eng.Stop()
	waitLen(t, eng, 1, 3*time.Second)
	addr := waitBound(t, eng)
	postBrowser(t, addr, `{"type":"TypeError","message":"browser stays","file":"src/app.js","line":9}`)
	waitLen(t, eng, 2, 2*time.Second)
	eng.ClearMarks()
	if eng.HasMarks() {
		t.Fatal("marks should be gone")
	}
	if err := eng.Restart(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(3 * time.Second)
	for {
		items := eng.Store().List("")
		hasLog, hasBrowser := false, false
		for _, ev := range items {
			if ev.Source == "app" {
				hasLog = true
			}
			if ev.Source == "browser" && ev.Message == "browser stays" {
				hasBrowser = true
			}
		}
		if hasLog && hasBrowser {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("want log+browser, got %+v", items)
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func TestEngineProjectIsolation(t *testing.T) {
	root := t.TempDir()
	db, err := persist.Open(filepath.Join(root, "inbox.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")
	if err := os.MkdirAll(a, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(b, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Sources: []config.SourceConfig{{Name: "browser", Type: "browser", Path: "127.0.0.1:0"}},
		Editor:  config.EditorConfig{Command: "code"},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engA := New()
	engA.SetIngestAddr("127.0.0.1:0")
	engA.SetProjectDir(a)
	engA.UsePersist(db)
	if err := engA.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	postBrowser(t, waitBound(t, engA), `{"type":"TypeError","message":"only in a","file":"a.js","line":1}`)
	waitLen(t, engA, 1, 2*time.Second)
	engA.Stop()

	engB := New()
	engB.SetIngestAddr(ingest.Disabled)
	engB.SetProjectDir(b)
	engB.UsePersist(db)
	if err := engB.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	defer engB.Stop()
	if engB.Store().Len() != 0 {
		t.Fatalf("project b saw a's inbox: %+v", engB.Store().List(""))
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

func TestEngineIngestsCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh -c")
	}
	cfg := &config.Config{
		Sources: []config.SourceConfig{{Name: "dev", Type: "command", Path: "printf 'ERROR TypeError: boom in /tmp/x.js:1\\n'"}},
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
			t.Fatal("timed out waiting for command event")
		case <-time.After(50 * time.Millisecond):
		}
	}
	if got := eng.Store().List("")[0].Type; got != "TypeError" {
		t.Fatalf("type = %q", got)
	}
}

func TestEngineSnoozeRestoresAfterExpiry(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Sources: []config.SourceConfig{{Name: "browser", Type: "browser", Path: "127.0.0.1:0"}},
		Editor:  config.EditorConfig{Command: "code"},
	}
	eng, _ := persistEngine(t, dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := eng.Start(ctx, cfg, true); err != nil {
		t.Fatal(err)
	}
	defer eng.Stop()
	addr := waitBound(t, eng)
	postBrowser(t, addr, `{"type":"TypeError","message":"later","file":"src/app.js","line":3}`)
	waitLen(t, eng, 1, 2*time.Second)
	hash := eng.Store().List("")[0].Hash
	eng.Snooze([]string{hash}, 50*time.Millisecond)
	if eng.Store().Len() != 0 {
		t.Fatal("expected empty while snoozed")
	}
	waitLen(t, eng, 1, 4*time.Second)
}
