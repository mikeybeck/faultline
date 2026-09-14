package parser

import (
	"strings"
	"testing"

	"github.com/mikey/faultline/internal/event"
)

func TestParseJSONPino(t *testing.T) {
	line := `{"level":50,"time":1710000000000,"pid":1,"hostname":"dev","msg":"Cannot charge","err":{"type":"TypeError","message":"Cannot charge","stack":"TypeError: Cannot charge\n    at charge (/app/src/pay.js:81:1)"}}`
	ev := ParseJSONLine("app", line)
	if ev == nil {
		t.Fatal("expected event")
	}
	if ev.Severity != event.SeverityError {
		t.Fatalf("severity = %q", ev.Severity)
	}
	if ev.Type != "TypeError" {
		t.Fatalf("type = %q", ev.Type)
	}
	if ev.Message != "Cannot charge" {
		t.Fatalf("message = %q", ev.Message)
	}
	if !strings.HasSuffix(ev.File, "pay.js") {
		t.Fatalf("file = %q", ev.File)
	}
	if ev.Line != 81 {
		t.Fatalf("line = %d", ev.Line)
	}
}

func TestParseJSONWinston(t *testing.T) {
	line := `{"level":"error","message":"failed lookup","timestamp":"2026-07-15T14:31:01Z"}`
	ev := ParseJSONLine("app", line)
	if ev == nil {
		t.Fatal("expected event")
	}
	if ev.Severity != event.SeverityError {
		t.Fatalf("severity = %q", ev.Severity)
	}
	if ev.Message != "failed lookup" {
		t.Fatalf("message = %q", ev.Message)
	}
}

func TestParseJSONZapWarn(t *testing.T) {
	line := `{"level":"warn","ts":1710000000,"msg":"slow query","file":"db.go","line":12}`
	ev := ParseJSONLine("app", line)
	if ev == nil {
		t.Fatal("expected event")
	}
	if ev.Severity != event.SeverityWarning {
		t.Fatalf("severity = %q", ev.Severity)
	}
	if ev.File != "db.go" || ev.Line != 12 {
		t.Fatalf("loc = %s:%d", ev.File, ev.Line)
	}
}

func TestParseJSONSkipsInfo(t *testing.T) {
	if ev := ParseJSONLine("app", `{"level":30,"msg":"hello"}`); ev != nil {
		t.Fatalf("info should be skipped: %+v", ev)
	}
	if ev := ParseJSONLine("app", `{"level":"debug","message":"x"}`); ev != nil {
		t.Fatalf("debug should be skipped: %+v", ev)
	}
}

func TestParseJSONIgnoresPlainText(t *testing.T) {
	if ev := ParseJSONLine("app", "ERROR boom"); ev != nil {
		t.Fatalf("plain text: %+v", ev)
	}
}

func TestJSONParserFeed(t *testing.T) {
	p := NewJSON("api")
	got := p.Feed(`{"level":"error","message":"nope"}`)
	if len(got) != 1 || got[0].Source != "api" {
		t.Fatalf("got %+v", got)
	}
}
