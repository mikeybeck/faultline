package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenericParseSample(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "generic_sample.log"))
	if err != nil {
		t.Fatal(err)
	}
	p := NewGeneric("app")
	var events []*struct {
		Type     string
		File     string
		Line     int
		Severity string
	}
	for _, line := range strings.Split(string(data), "\n") {
		for _, ev := range p.Feed(line) {
			events = append(events, &struct {
				Type     string
				File     string
				Line     int
				Severity string
			}{ev.Type, ev.File, ev.Line, string(ev.Severity)})
		}
	}
	for _, ev := range p.Flush() {
		events = append(events, &struct {
			Type     string
			File     string
			Line     int
			Severity string
		}{ev.Type, ev.File, ev.Line, string(ev.Severity)})
	}

	if len(events) < 4 {
		t.Fatalf("expected at least 4 events, got %d", len(events))
	}
	if events[0].Type != "TypeError" {
		t.Errorf("event0 type = %q, want TypeError", events[0].Type)
	}
	if !strings.HasSuffix(events[0].File, "payments.js") {
		t.Errorf("event0 file = %q, want payments.js", events[0].File)
	}
	if events[0].Line != 81 {
		t.Errorf("event0 line = %d, want 81", events[0].Line)
	}
	if events[1].Type != "IntegrityError" {
		t.Errorf("event1 type = %q, want IntegrityError", events[1].Type)
	}
	if !strings.HasSuffix(events[1].File, "views.py") && !strings.HasSuffix(events[1].File, "models.py") {
		t.Errorf("event1 file = %q, want views.py or models.py", events[1].File)
	}
	if events[2].Type != "panic" && !strings.EqualFold(events[2].Type, "panic") {
		t.Errorf("event2 type = %q, want panic", events[2].Type)
	}
	if !strings.HasSuffix(events[2].File, "main.go") {
		t.Errorf("event2 file = %q, want main.go", events[2].File)
	}
	if events[2].Line != 19 && events[2].Line != 8 {
		t.Errorf("event2 line = %d, want 19 or 8", events[2].Line)
	}
	foundWarn := false
	for _, ev := range events {
		if ev.Severity == "warning" {
			foundWarn = true
		}
	}
	if !foundWarn {
		t.Fatal("expected a warning event")
	}
}

func TestGenericIgnoresInfoDebug(t *testing.T) {
	p := NewGeneric("app")
	p.Feed("INFO started worker pool")
	p.Feed("DEBUG cache hit for session")
	if got := p.Flush(); len(got) != 0 {
		t.Fatalf("expected info/debug to be ignored, got %#v", got)
	}
}

func TestForTypeGeneric(t *testing.T) {
	p, err := ForType("generic", "app")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.(*Generic); !ok {
		t.Fatalf("got %T", p)
	}
}
