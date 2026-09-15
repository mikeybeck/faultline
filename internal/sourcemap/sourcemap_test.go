package sourcemap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikey/faultline/internal/event"
)

func TestLookupGeneratedLine(t *testing.T) {
	// generated line 2 col 0 -> src/App.svelte:81
	mappings := ";" + encodeVLQ(0) + encodeVLQ(0) + encodeVLQ(80) + encodeVLQ(0)
	raw, _ := json.Marshal(map[string]any{
		"version":  3,
		"sources":  []string{"src/App.svelte"},
		"mappings": mappings,
	})
	m, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	file, line, _, ok := m.Lookup(2, 0)
	if !ok {
		t.Fatal("lookup failed")
	}
	if file != "src/App.svelte" || line != 81 {
		t.Fatalf("got %s:%d", file, line)
	}
}

func TestResolverReadsSiblingMap(t *testing.T) {
	dir := t.TempDir()
	js := filepath.Join(dir, "bundle.js")
	if err := os.WriteFile(js, []byte("function Yt(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mappings := encodeVLQ(0) + encodeVLQ(0) + encodeVLQ(9) + encodeVLQ(0)
	raw, _ := json.Marshal(map[string]any{
		"version":  3,
		"sources":  []string{"src/lib.ts"},
		"mappings": mappings,
	})
	if err := os.WriteFile(js+".map", raw, 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewResolver(dir)
	file, line, ok := r.Remap(js, 1, 0)
	if !ok {
		t.Fatal("remap failed")
	}
	want := filepath.Join(dir, "src/lib.ts")
	if file != "src/lib.ts" && file != want {
		t.Fatalf("file = %q", file)
	}
	if line != 10 {
		t.Fatalf("line = %d", line)
	}
}

func TestApplyRewritesEvent(t *testing.T) {
	dir := t.TempDir()
	js := filepath.Join(dir, "index-BMpqHCu2.js")
	if err := os.WriteFile(js, []byte("Yt()\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mappings := encodeVLQ(0) + encodeVLQ(0) + encodeVLQ(82) + encodeVLQ(0)
	raw, _ := json.Marshal(map[string]any{
		"version":  3,
		"sources":  []string{"src/ProjectForm.svelte"},
		"mappings": mappings,
	})
	if err := os.WriteFile(js+".map", raw, 0o644); err != nil {
		t.Fatal(err)
	}
	ev := event.Event{
		Source:  "browser",
		Type:    "ReferenceError",
		Message: "line is not defined",
		File:    js,
		Line:    1,
		Stack:   "ReferenceError: line is not defined\n    at Yt (" + js + ":1:0)",
	}
	Apply(NewResolver(dir), &ev, 0)
	if !strings.Contains(ev.File, "ProjectForm.svelte") {
		t.Fatalf("file = %q", ev.File)
	}
	if ev.Line != 83 {
		t.Fatalf("line = %d", ev.Line)
	}
	if !strings.Contains(ev.Stack, "ProjectForm.svelte:83") {
		t.Fatalf("stack = %q", ev.Stack)
	}
}
