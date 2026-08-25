package store

import (
	"testing"
	"time"

	"github.com/mikey/faultline/internal/event"
)

func TestIngestGroupsDuplicates(t *testing.T) {
	s := New()
	base := event.Event{
		Source:   "laravel",
		Type:     "TypeError",
		Message:  "bad arg",
		File:     "PaymentController.php",
		Line:     81,
		Time:     time.Now(),
		Severity: event.SeverityError,
	}
	base.Hash = event.Fingerprint(base.Source, base.Type, base.Message, base.File, base.Line)

	r1 := s.Ingest(base)
	if !r1.IsNew || r1.Event.Count != 1 {
		t.Fatalf("first ingest: IsNew=%v Count=%d", r1.IsNew, r1.Event.Count)
	}
	base.Time = base.Time.Add(time.Second)
	r2 := s.Ingest(base)
	if r2.IsNew || r2.Event.Count != 2 {
		t.Fatalf("second ingest: IsNew=%v Count=%d", r2.IsNew, r2.Event.Count)
	}
	if s.Len() != 1 {
		t.Fatalf("Len = %d, want 1", s.Len())
	}
}

func TestListFilterAndClear(t *testing.T) {
	s := New()
	s.Ingest(event.Event{Source: "a", Type: "TypeError", Message: "x", File: "A.php", Line: 1, Time: time.Now()})
	s.Ingest(event.Event{Source: "a", Type: "SQLSTATE[23000]", Message: "dup", File: "B.php", Line: 2, Time: time.Now()})
	if got := s.List("sqlstate"); len(got) != 1 {
		t.Fatalf("filter sqlstate => %d", len(got))
	}
	s.Clear()
	if s.Len() != 0 {
		t.Fatal("expected empty after clear")
	}
}

func TestByFrequency(t *testing.T) {
	s := New()
	a := event.Event{Source: "a", Type: "A", Message: "a", File: "a.php", Line: 1, Time: time.Now()}
	b := event.Event{Source: "a", Type: "B", Message: "b", File: "b.php", Line: 2, Time: time.Now()}
	s.Ingest(a)
	s.Ingest(b)
	s.Ingest(b)
	s.Ingest(b)
	items := s.ByFrequency("")
	if len(items) != 2 || items[0].Type != "B" || items[0].Count != 3 {
		t.Fatalf("unexpected frequency order: %+v", items)
	}
}

func TestListOrdersByLastSeen(t *testing.T) {
	s := New()
	t0 := time.Now()
	a := event.Event{Source: "a", Type: "A", Message: "a", File: "a.php", Line: 1, Time: t0}
	b := event.Event{Source: "a", Type: "B", Message: "b", File: "b.php", Line: 2, Time: t0.Add(time.Second)}
	s.Ingest(a)
	s.Ingest(b)
	a.Time = t0.Add(2 * time.Second)
	s.Ingest(a)
	items := s.List("")
	if len(items) != 2 || items[0].Type != "A" {
		t.Fatalf("expected A first after later LastSeen, got %+v", items)
	}
}

func TestSummariesOmitsStackAndRaw(t *testing.T) {
	s := New()
	s.Ingest(event.Event{
		Source:  "a",
		Type:    "TypeError",
		Message: "x",
		File:    "A.php",
		Line:    1,
		Time:    time.Now(),
		Stack:   "stack dump",
		Raw:     "raw dump",
	})
	items := s.Summaries("")
	if len(items) != 1 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0].Stack != "" || items[0].Raw != "" {
		t.Fatalf("summaries should omit stack/raw: %+v", items[0])
	}
	full, ok := s.Get(items[0].Hash)
	if !ok || full.Stack != "stack dump" || full.Raw != "raw dump" {
		t.Fatalf("Get should keep stack/raw: %+v", full)
	}
}
