package main

import (
	"testing"

	"github.com/mikey/faultline/internal/event"
)

func TestToSummaryDTOOmitsStackAndRaw(t *testing.T) {
	ev := event.Event{Type: "TypeError", Stack: "stack", Raw: "raw", Hash: "abc"}
	full := toDTO(ev, "")
	if full.Stack != "stack" || full.Raw != "raw" {
		t.Fatalf("toDTO dropped detail: %+v", full)
	}
	sum := toSummaryDTO(ev)
	if sum.Stack != "" || sum.Raw != "" {
		t.Fatalf("summary should omit stack/raw: %+v", sum)
	}
	if sum.Type != "TypeError" || sum.Hash != "abc" {
		t.Fatalf("summary dropped identity: %+v", sum)
	}
}
