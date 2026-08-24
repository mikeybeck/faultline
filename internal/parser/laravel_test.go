package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLaravelParseSample(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "laravel_sample.log"))
	if err != nil {
		t.Fatal(err)
	}
	p := NewLaravel("laravel")
	var events []*struct {
		Type string
		File string
		Line int
	}
	for _, line := range strings.Split(string(data), "\n") {
		for _, ev := range p.Feed(line) {
			events = append(events, &struct {
				Type string
				File string
				Line int
			}{ev.Type, ev.File, ev.Line})
		}
	}
	for _, ev := range p.Flush() {
		events = append(events, &struct {
			Type string
			File string
			Line int
		}{ev.Type, ev.File, ev.Line})
	}

	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(events))
	}
	if events[0].Type != "TypeError" {
		t.Errorf("event0 type = %q, want TypeError", events[0].Type)
	}
	if !strings.HasSuffix(events[0].File, "PaymentController.php") {
		t.Errorf("event0 file = %q, want PaymentController.php", events[0].File)
	}
	if events[0].Line != 81 {
		t.Errorf("event0 line = %d, want 81", events[0].Line)
	}
	if !strings.HasPrefix(events[1].Type, "SQLSTATE") && events[1].Type != "Illuminate\\Database\\QueryException" {
		// Message starts with SQLSTATE; type may be SQLSTATE[23000]
		if !strings.Contains(events[1].Type, "SQLSTATE") {
			t.Errorf("event1 type = %q, want SQLSTATE…", events[1].Type)
		}
	}
	if !strings.HasSuffix(events[1].File, "User.php") {
		t.Errorf("event1 file = %q, want User.php", events[1].File)
	}
	if events[1].Line != 41 {
		t.Errorf("event1 line = %d, want 41", events[1].Line)
	}
	if events[2].Type != "ErrorException" {
		t.Errorf("event2 type = %q, want ErrorException", events[2].Type)
	}
}

func TestLaravelIgnoresInfo(t *testing.T) {
	p := NewLaravel("laravel")
	p.Feed(`[2026-07-15 14:33:00] local.INFO: User logged in {"id":1}`)
	if got := p.Flush(); len(got) != 0 {
		t.Fatalf("expected info to be ignored, got %#v", got)
	}
}

func TestLaravelDedupFingerprintStable(t *testing.T) {
	p := NewLaravel("laravel")
	lines := []string{
		`[2026-07-15 14:31:01] local.ERROR: TypeError: boom in PaymentController.php on line 81`,
		``,
		`[2026-07-15 14:31:02] local.ERROR: TypeError: boom in PaymentController.php on line 81`,
		``,
	}
	var hashes []string
	for _, line := range lines {
		for _, ev := range p.Feed(line) {
			hashes = append(hashes, ev.Hash)
		}
	}
	for _, ev := range p.Flush() {
		hashes = append(hashes, ev.Hash)
	}
	if len(hashes) != 2 {
		t.Fatalf("expected 2 events, got %d", len(hashes))
	}
	if hashes[0] != hashes[1] {
		t.Fatalf("hashes differ for duplicate errors: %s vs %s", hashes[0], hashes[1])
	}
}
