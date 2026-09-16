package sourcemap

import (
	"encoding/base64"
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

func TestResolverIgnoresMapOutsideProject(t *testing.T) {
	dir := t.TempDir()
	js := filepath.Join(dir, "bundle.js")
	if err := os.WriteFile(js, []byte("//# sourceMappingURL=../../secret.map\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(filepath.Dir(dir), "secret.map")
	raw, _ := json.Marshal(map[string]any{
		"version":  3,
		"sources":  []string{"stolen.ts"},
		"mappings": encodeVLQ(0) + encodeVLQ(0) + encodeVLQ(0) + encodeVLQ(0),
	})
	if err := os.WriteFile(secret, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(secret) })
	r := NewResolver(dir)
	if _, _, ok := r.Remap(js, 1, 0); ok {
		t.Fatal("should not follow sourceMappingURL outside the project")
	}
}

func TestIsLocalURL(t *testing.T) {
	if !isLocalURL("http://127.0.0.1:5173/app.js") {
		t.Fatal("loopback should be local")
	}
	if isLocalURL("http://example.com/app.js.map") {
		t.Fatal("remote host should be rejected")
	}
	if isLocalURL("http://169.254.169.254/latest/meta-data") {
		t.Fatal("link-local should be rejected")
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

func TestInlineDataMapAndSourcesContent(t *testing.T) {
	dir := t.TempDir()
	src := "export function boom() {\n  throw new Error('x')\n}\n"
	mappings := encodeVLQ(0) + encodeVLQ(0) + encodeVLQ(1) + encodeVLQ(0)
	raw, _ := json.Marshal(map[string]any{
		"version":        3,
		"sources":        []string{"src/lib.ts"},
		"sourcesContent": []string{src},
		"mappings":       mappings,
	})
	b64 := base64.StdEncoding.EncodeToString(raw)
	js := filepath.Join(dir, "bundle.js")
	if err := os.WriteFile(js, []byte("//# sourceMappingURL=data:application/json;base64,"+b64+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewResolver(dir)
	file, line, ok := r.Remap(js, 1, 0)
	if !ok {
		t.Fatal("remap failed")
	}
	if !strings.Contains(file, "lib.ts") {
		t.Fatalf("file=%q", file)
	}
	if line != 2 {
		t.Fatalf("line=%d", line)
	}
	sn := r.snippetFromGenerated(js, 1, 0)
	if !strings.Contains(sn, "throw new Error") {
		t.Fatalf("snippet=%q", sn)
	}
}
