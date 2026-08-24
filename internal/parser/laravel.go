package parser

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mikey/faultline/internal/event"
)

// Laravel log header examples:
// [2026-07-15 14:31:01] local.ERROR: TypeError: ...
// [2026-07-15 14:31:01] production.ERROR: SQLSTATE[23000]: ...

var (
	laravelHeaderRe = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?)\]\s+(\w+)\.(\w+):\s*(.*)$`)
	laravelStackRe  = regexp.MustCompile(`^#\d+\s+(.*)$`)
	laravelFrameRe  = regexp.MustCompile(`^(.*?)\((\d+)\):\s*(.*)$`)
	laravelAtFileRe = regexp.MustCompile(`(?i)\bin\s+([^\s:]+(?:\.php))\s*(?:on\s+line\s+|:)(\d+)`)
	laravelTypeRe   = regexp.MustCompile(`^([A-Za-z0-9_\\]+(?:Error|Exception|Throwable))(?::\s*(.*))?$`)
)

type Laravel struct {
	source  string
	current *pendingLaravel
}

type pendingLaravel struct {
	headerTime time.Time
	env        string
	level      string
	message    string
	stackLines []string
	raw        strings.Builder
}

func NewLaravel(source string) *Laravel {
	return &Laravel{source: source}
}

func (p *Laravel) Feed(line string) []*event.Event {
	if m := laravelHeaderRe.FindStringSubmatch(line); m != nil {
		var out []*event.Event
		if p.current != nil {
			if ev := p.finish(); ev != nil {
				out = append(out, ev)
			}
		}
		ts, _ := parseLaravelTime(m[1])
		p.current = &pendingLaravel{
			headerTime: ts,
			env:        m[2],
			level:      m[3],
			message:    m[4],
		}
		p.current.raw.WriteString(line)
		return out
	}

	if p.current == nil {
		return nil
	}

	trimmed := strings.TrimSpace(line)

	// Blank line after content ends the record for live streaming.
	if trimmed == "" {
		if ev := p.finish(); ev != nil {
			return []*event.Event{ev}
		}
		return nil
	}

	p.current.raw.WriteByte('\n')
	p.current.raw.WriteString(line)

	if laravelStackRe.MatchString(trimmed) || strings.HasPrefix(trimmed, "[stacktrace]") || trimmed == "Stack trace:" {
		p.current.stackLines = append(p.current.stackLines, line)
	} else if len(p.current.stackLines) > 0 && (strings.HasPrefix(trimmed, "#") || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || strings.HasPrefix(trimmed, "}")) {
		p.current.stackLines = append(p.current.stackLines, line)
		// Closing brace often ends Laravel's JSON-ish stack payload.
		if trimmed == "}\"" || trimmed == "}" {
			if ev := p.finish(); ev != nil {
				return []*event.Event{ev}
			}
		}
	} else if !strings.HasPrefix(trimmed, "{") {
		// Continuation of message (pre-stack).
		if len(p.current.stackLines) == 0 {
			p.current.message += " " + trimmed
		}
	}
	return nil
}

func (p *Laravel) Flush() []*event.Event {
	if p.current == nil {
		return nil
	}
	ev := p.finish()
	if ev == nil {
		return nil
	}
	return []*event.Event{ev}
}

func (p *Laravel) finish() *event.Event {
	cur := p.current
	p.current = nil
	if cur == nil {
		return nil
	}

	level := strings.ToUpper(cur.level)
	// Only surface error-ish levels for the inbox.
	switch level {
	case "ERROR", "CRITICAL", "ALERT", "EMERGENCY", "WARNING":
	default:
		return nil
	}

	typ, msg := splitLaravelTypeMessage(cur.message)
	file, line := firstAppFrame(cur.stackLines)
	if file == "" {
		file, line = fileFromMessage(cur.message)
	}

	sev := event.SeverityError
	switch level {
	case "WARNING":
		sev = event.SeverityWarning
	case "CRITICAL", "ALERT", "EMERGENCY":
		sev = event.SeverityCritical
	}

	now := cur.headerTime
	if now.IsZero() {
		now = time.Now()
	}

	hash := event.Fingerprint(p.source, typ, msg, filepath.Base(file), line)
	return &event.Event{
		Source:    p.source,
		Time:      now,
		Type:      typ,
		Message:   msg,
		File:      file,
		Line:      line,
		Severity:  sev,
		Stack:     strings.Join(cur.stackLines, "\n"),
		Raw:       cur.raw.String(),
		Hash:      hash,
		Count:     1,
		FirstSeen: now,
		LastSeen:  now,
	}
}

func parseLaravelTime(s string) (time.Time, error) {
	s = strings.ReplaceAll(s, "T", " ")
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000000",
		"2006-01-02 15:04:05.000",
	}
	var last error
	for _, f := range formats {
		t, err := time.ParseInLocation(f, s, time.Local)
		if err == nil {
			return t, nil
		}
		last = err
	}
	return time.Time{}, last
}

func splitLaravelTypeMessage(message string) (typ, msg string) {
	message = strings.TrimSpace(message)
	if m := laravelTypeRe.FindStringSubmatch(message); m != nil {
		typ = m[1]
		msg = strings.TrimSpace(m[2])
		if msg == "" {
			msg = message
		}
		return typ, msg
	}
	// SQLSTATE[...]
	if strings.HasPrefix(message, "SQLSTATE[") {
		if idx := strings.Index(message, ":"); idx > 0 {
			return strings.TrimSpace(message[:idx]), strings.TrimSpace(message)
		}
		return "SQLSTATE", message
	}
	// Fall back to first token before colon.
	if idx := strings.Index(message, ":"); idx > 0 && idx < 80 {
		head := strings.TrimSpace(message[:idx])
		if !strings.Contains(head, " ") {
			return head, strings.TrimSpace(message[idx+1:])
		}
	}
	return "Error", message
}

func firstAppFrame(stack []string) (string, int) {
	var fallbackFile string
	var fallbackLine int
	for _, line := range stack {
		trimmed := strings.TrimSpace(line)
		m := laravelStackRe.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}
		body := m[1]
		fm := laravelFrameRe.FindStringSubmatch(body)
		if fm == nil {
			continue
		}
		file := fm[1]
		ln, _ := strconv.Atoi(fm[2])
		if fallbackFile == "" {
			fallbackFile = file
			fallbackLine = ln
		}
		if isAppFrame(file) {
			return file, ln
		}
	}
	return fallbackFile, fallbackLine
}

func isAppFrame(file string) bool {
	norm := filepath.ToSlash(file)
	if strings.Contains(norm, "/vendor/") || strings.HasPrefix(norm, "vendor/") {
		return false
	}
	if strings.Contains(norm, "/Illuminate/") || strings.Contains(norm, "/laravel/framework/") {
		return false
	}
	base := filepath.Base(file)
	return strings.HasSuffix(strings.ToLower(base), ".php")
}

func fileFromMessage(message string) (string, int) {
	if m := laravelAtFileRe.FindStringSubmatch(message); m != nil {
		ln, _ := strconv.Atoi(m[2])
		return m[1], ln
	}
	return "", 0
}
