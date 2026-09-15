package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikey/faultline/internal/event"
)

func TestToEventParsesViteStack(t *testing.T) {
	p := Payload{
		Type:     "TypeError",
		Message:  "Cannot read properties of null",
		Stack:    "TypeError: Cannot read properties of null\n    at checkout (http://localhost:5173/src/payments.js:81:12)",
		URL:      "http://localhost:5173/checkout",
		Severity: "error",
	}
	ev := ToEvent("browser", "", p)
	if ev.Type != "TypeError" {
		t.Fatalf("type = %q", ev.Type)
	}
	if ev.File != "src/payments.js" {
		t.Fatalf("file = %q", ev.File)
	}
	if ev.Line != 81 {
		t.Fatalf("line = %d", ev.Line)
	}
	if ev.Severity != event.SeverityError {
		t.Fatalf("severity = %q", ev.Severity)
	}
	if !strings.Contains(ev.Raw, "checkout") {
		t.Fatalf("raw missing url: %q", ev.Raw)
	}
}

func TestToEventPrefersPayloadFile(t *testing.T) {
	p := Payload{
		Type:    "ReferenceError",
		Message: "x is not defined",
		File:    "http://localhost:3000/app.ts",
		Line:    12,
		Stack:   "at other (http://localhost:3000/other.ts:99:1)",
	}
	ev := ToEvent("browser", "", p)
	if ev.File != "app.ts" {
		t.Fatalf("file = %q", ev.File)
	}
	if ev.Line != 12 {
		t.Fatalf("line = %d", ev.Line)
	}
}

func TestResolveFileJoinsProject(t *testing.T) {
	dir := t.TempDir()
	rel := filepath.Join("src", "app.js")
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ResolveFile("http://localhost:5173/src/app.js", dir)
	if got != full {
		t.Fatalf("got %q want %q", got, full)
	}
}

func TestResolveFileViteFS(t *testing.T) {
	dir := t.TempDir()
	full := filepath.Join(dir, "lib.ts")
	if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ResolveFile("http://localhost:5173/@fs"+full, dir)
	if got != full {
		t.Fatalf("got %q want %q", got, full)
	}
}

func TestResolveFileRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(filepath.Dir(dir), "secret-outside.js")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(outside) })
	got := ResolveFile("../"+filepath.Base(outside), dir)
	if got == outside {
		t.Fatalf("resolved file outside project: %q", got)
	}
}

func TestResolveFileByBasename(t *testing.T) {
	dir := t.TempDir()
	full := filepath.Join(dir, "src", "components", "Pay.js")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ResolveFile("http://localhost:5173/assets/Pay.js", dir)
	if got != full {
		t.Fatalf("got %q want %q", got, full)
	}
}

func TestParseFramesResolves(t *testing.T) {
	frames := ParseFrames("TypeError: x\n    at checkout (http://localhost:5173/src/payments.js:81:12)", "")
	if len(frames) != 2 {
		t.Fatalf("frames = %d", len(frames))
	}
	if frames[1].File != "src/payments.js" || frames[1].Line != 81 {
		t.Fatalf("frame = %+v", frames[1])
	}
}

func TestToEventSkipsNoisyFrames(t *testing.T) {
	p := Payload{
		Type:    "Error",
		Message: "boom",
		Stack:   "Error: boom\n    at http://localhost:5173/node_modules/vue.js:1:1\n    at http://localhost:5173/src/App.vue:10:2",
	}
	ev := ToEvent("browser", "", p)
	if ev.File != "src/App.vue" {
		t.Fatalf("file = %q", ev.File)
	}
	if ev.Line != 10 {
		t.Fatalf("line = %d", ev.Line)
	}
}
