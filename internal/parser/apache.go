package parser

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mikey/faultline/internal/event"
)

// Common Apache error log formats:
// [Wed Jul 15 14:31:01.123456 2026] [php:error] [pid 123] [client 127.0.0.1:54321] PHP Fatal error: ... in /path/file.php on line 82
// [Wed Jul 15 14:31:01 2026] [error] [client 1.2.3.4] File does not exist: /var/www/...

var (
	apacheHeaderRe = regexp.MustCompile(`^\[([^\]]+)\]\s+(?:\[([^\]]+)\]\s+)?(?:\[pid\s+[^\]]+\]\s+)?(?:\[client\s+[^\]]+\]\s+)?(.*)$`)
	apacheFileRe   = regexp.MustCompile(`(?i)(?:in|at)\s+(/[^\s:]+?\.(?:php|js|py|go|rb|ts|tsx|jsx))\s*(?:on\s+line\s+|:)(\d+)`)
	apacheFileRe2  = regexp.MustCompile(`(?i)(/[^\s:]+?\.(?:php|js|py|go|rb))\s+on\s+line\s+(\d+)`)
	phpTypeRe      = regexp.MustCompile(`(?i)^(?:PHP\s+)?((?:Fatal\s+error|Parse\s+error|Warning|Notice|Deprecated|TypeError|Error|Exception))\s*:\s*(.*)$`)
)

type Apache struct {
	source string
}

func NewApache(source string) *Apache {
	return &Apache{source: source}
}

func (p *Apache) Feed(line string) []*event.Event {
	line = strings.TrimRight(line, "\r")
	if strings.TrimSpace(line) == "" {
		return nil
	}
	m := apacheHeaderRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	ts, _ := parseApacheTime(m[1])
	moduleLevel := m[2]
	message := strings.TrimSpace(m[3])
	if message == "" {
		return nil
	}

	sev, module := parseApacheSeverity(moduleLevel, message)
	// Skip pure info/debug noise.
	if sev == event.SeverityInfo && !looksLikeError(message) {
		return nil
	}

	typ, msg := splitApacheTypeMessage(message)
	file, ln := extractApacheFileLine(message)

	if ts.IsZero() {
		ts = time.Now()
	}

	hash := event.Fingerprint(p.source, typ, msg, filepath.Base(file), ln)
	return []*event.Event{{
		Source:    p.source,
		Time:      ts,
		Type:      typ,
		Message:   msg,
		File:      file,
		Line:      ln,
		Severity:  sev,
		Raw:       line,
		Hash:      hash,
		Count:     1,
		FirstSeen: ts,
		LastSeen:  ts,
		Stack:     module,
	}}
}

func (p *Apache) Flush() []*event.Event {
	return nil
}

func parseApacheTime(s string) (time.Time, error) {
	formats := []string{
		"Mon Jan 02 15:04:05.000000 2006",
		"Mon Jan 2 15:04:05.000000 2006",
		"Mon Jan 02 15:04:05 2006",
		"Mon Jan 2 15:04:05 2006",
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

func parseApacheSeverity(moduleLevel, message string) (event.Severity, string) {
	module := ""
	level := strings.ToLower(moduleLevel)
	if idx := strings.Index(moduleLevel, ":"); idx >= 0 {
		module = moduleLevel[:idx]
		level = strings.ToLower(moduleLevel[idx+1:])
	}
	switch {
	case strings.Contains(level, "emerg"), strings.Contains(level, "alert"), strings.Contains(level, "crit"):
		return event.SeverityCritical, module
	case strings.Contains(level, "error"), strings.Contains(level, "fatal"):
		return event.SeverityError, module
	case strings.Contains(level, "warn"):
		return event.SeverityWarning, module
	}
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "fatal"), strings.Contains(lower, "emergency"):
		return event.SeverityCritical, module
	case strings.Contains(lower, "error"), strings.Contains(lower, "exception"):
		return event.SeverityError, module
	case strings.Contains(lower, "warn"), strings.Contains(lower, "deprecated"), strings.Contains(lower, "notice"):
		return event.SeverityWarning, module
	case strings.Contains(lower, "denied"), strings.Contains(lower, "forbidden"),
		strings.Contains(lower, "unauthorized"):
		return event.SeverityError, module
	default:
		return event.SeverityInfo, module
	}
}

func looksLikeError(message string) bool {
	lower := strings.ToLower(message)
	keys := []string{"error", "exception", "fatal", "failed", "warning", "denied", "segfault", "forbidden", "unauthorized"}
	for _, k := range keys {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

func splitApacheTypeMessage(message string) (typ, msg string) {
	if m := phpTypeRe.FindStringSubmatch(message); m != nil {
		typ = strings.TrimSpace(m[1])
		// Normalize "Fatal error" style.
		typ = strings.ReplaceAll(typ, " ", "")
		if strings.EqualFold(typ, "Fatalerror") {
			typ = "PHP Fatal error"
		} else if strings.EqualFold(typ, "Parseerror") {
			typ = "PHP Parse error"
		}
		msg = strings.TrimSpace(m[2])
		// Strip trailing "in file on line N"
		if idx := strings.LastIndex(strings.ToLower(msg), " in /"); idx > 0 {
			msg = strings.TrimSpace(msg[:idx])
		}
		return typ, msg
	}
	if strings.HasPrefix(message, "AH") {
		parts := strings.SplitN(message, ":", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		}
	}
	return "Apache", message
}

func extractApacheFileLine(message string) (string, int) {
	if m := apacheFileRe.FindStringSubmatch(message); m != nil {
		ln, _ := strconv.Atoi(m[2])
		return m[1], ln
	}
	if m := apacheFileRe2.FindStringSubmatch(message); m != nil {
		ln, _ := strconv.Atoi(m[2])
		return m[1], ln
	}
	return "", 0
}
