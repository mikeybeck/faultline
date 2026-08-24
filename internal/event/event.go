package event

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Severity ranks relative importance of an event.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityError    Severity = "error"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Event is a normalized log exception or error.
type Event struct {
	Source    string
	Time      time.Time
	Type      string
	Message   string
	File      string
	Line      int
	Severity  Severity
	Stack     string
	Raw       string
	Hash      string
	Count     int
	FirstSeen time.Time
	LastSeen  time.Time
}

// Location returns "file:line" when available.
func (e Event) Location() string {
	if e.File == "" {
		return ""
	}
	if e.Line <= 0 {
		return e.File
	}
	return fmt.Sprintf("%s:%d", e.File, e.Line)
}

// Title is a short label for list and notifications.
func (e Event) Title() string {
	if e.Type != "" {
		return e.Type
	}
	if e.Message != "" {
		return truncate(e.Message, 60)
	}
	return "Unknown error"
}

// Fingerprint builds a stable hash used for grouping duplicates.
func Fingerprint(source, typ, message, file string, line int) string {
	var b strings.Builder
	b.WriteString(strings.ToLower(strings.TrimSpace(source)))
	b.WriteByte('|')
	b.WriteString(strings.ToLower(strings.TrimSpace(typ)))
	b.WriteByte('|')
	b.WriteString(normalizeMessage(message))
	b.WriteByte('|')
	b.WriteString(strings.TrimSpace(file))
	b.WriteByte('|')
	b.WriteString(fmt.Sprintf("%d", line))
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:16])
}

func normalizeMessage(msg string) string {
	msg = strings.ToLower(strings.TrimSpace(msg))
	// Collapse runs of whitespace.
	fields := strings.Fields(msg)
	return strings.Join(fields, " ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}
