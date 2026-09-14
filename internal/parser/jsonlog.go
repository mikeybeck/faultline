package parser

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mikey/faultline/internal/event"
)

// JSON parses newline-delimited JSON logs (Pino, Winston, Zap, and similar).
type JSON struct {
	source string
}

func NewJSON(source string) *JSON {
	return &JSON{source: source}
}

func (p *JSON) Feed(line string) []*event.Event {
	ev := ParseJSONLine(p.source, strings.TrimSpace(strings.TrimRight(line, "\r")))
	if ev == nil {
		return nil
	}
	return []*event.Event{ev}
}

func (p *JSON) Flush() []*event.Event { return nil }

// ParseJSONLine turns a single JSON log object into an event.
// Info/debug lines and non-objects return nil.
func ParseJSONLine(source, line string) *event.Event {
	line = strings.TrimSpace(line)
	if len(line) < 2 || line[0] != '{' {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		return nil
	}
	sev, ok := jsonSeverity(m)
	if !ok {
		return nil
	}
	msg := jsonString(m, "msg", "message")
	typ := jsonString(m, "type", "name", "errorType")
	stack := jsonString(m, "stack", "stacktrace", "stack_trace")
	file := jsonString(m, "file", "filename", "path")
	lineNo := jsonInt(m, "line", "lineno", "lineNumber")

	if errObj, ok := nestedMap(m, "err", "error", "exception"); ok {
		if msg == "" {
			msg = jsonString(errObj, "message", "msg")
		}
		if typ == "" {
			typ = jsonString(errObj, "type", "name")
		}
		if stack == "" {
			stack = jsonString(errObj, "stack", "stacktrace")
		}
		if file == "" {
			file = jsonString(errObj, "file", "filename")
		}
		if lineNo == 0 {
			lineNo = jsonInt(errObj, "line", "lineno")
		}
	}
	if msg == "" {
		msg = jsonString(m, "error", "err")
	}
	if msg == "" {
		msg = line
	}
	if typ == "" {
		typ = genericTypeRe.FindString(msg)
	}
	if typ == "" {
		typ = "Error"
	}
	if file == "" || lineNo == 0 {
		f, l := extractGenericFileLine(msg + "\n" + stack)
		if file == "" {
			file = f
		}
		if lineNo == 0 {
			lineNo = l
		}
	}

	now := jsonTime(m)
	if now.IsZero() {
		now = time.Now()
	}
	return &event.Event{
		Source:    source,
		Time:      now,
		Type:      typ,
		Message:   msg,
		File:      file,
		Line:      lineNo,
		Severity:  sev,
		Stack:     stack,
		Raw:       line,
		Hash:      event.Fingerprint(source, typ, msg, filepath.Base(file), lineNo),
		Count:     1,
		FirstSeen: now,
		LastSeen:  now,
	}
}

func jsonSeverity(m map[string]any) (event.Severity, bool) {
	if v, ok := m["level"]; ok {
		switch n := v.(type) {
		case float64:
			if n >= 60 {
				return event.SeverityCritical, true
			}
			if n >= 50 {
				return event.SeverityError, true
			}
			if n >= 40 {
				return event.SeverityWarning, true
			}
			return event.SeverityInfo, false
		case json.Number:
			f, _ := n.Float64()
			return jsonSeverity(map[string]any{"level": f})
		}
	}
	level := strings.ToLower(jsonString(m, "level", "severity", "lvl"))
	switch level {
	case "fatal", "panic", "critical", "emerg", "emergency", "alert", "crit":
		return event.SeverityCritical, true
	case "error", "err", "exception":
		return event.SeverityError, true
	case "warn", "warning":
		return event.SeverityWarning, true
	case "info", "debug", "trace", "notice", "verbose":
		return event.SeverityInfo, false
	}
	if _, ok := nestedMap(m, "err", "error", "exception"); ok {
		return event.SeverityError, true
	}
	return event.SeverityInfo, false
}

func jsonString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch s := v.(type) {
			case string:
				if strings.TrimSpace(s) != "" {
					return s
				}
			case map[string]any:
				if msg := jsonString(s, "message", "msg", "stack"); msg != "" {
					return msg
				}
			}
		}
	}
	return ""
}

func jsonInt(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch n := v.(type) {
			case float64:
				return int(n)
			case json.Number:
				i, _ := n.Int64()
				return int(i)
			case string:
				i, _ := strconv.Atoi(n)
				return i
			}
		}
	}
	return 0
}

func jsonTime(m map[string]any) time.Time {
	for _, k := range []string{"time", "ts", "timestamp", "@timestamp"} {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch n := v.(type) {
		case string:
			for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
				if t, err := time.Parse(layout, n); err == nil {
					return t
				}
			}
		case float64:
			if n > 1e12 {
				return time.UnixMilli(int64(n))
			}
			if n > 1e9 {
				return time.Unix(int64(n), 0)
			}
		}
	}
	return time.Time{}
}

func nestedMap(m map[string]any, keys ...string) (map[string]any, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if obj, ok := v.(map[string]any); ok {
				return obj, true
			}
		}
	}
	return nil, false
}
