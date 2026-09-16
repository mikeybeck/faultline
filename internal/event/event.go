package event

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const maxSamples = 5

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
	Source    string    `json:"source"`
	Time      time.Time `json:"time"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	File      string    `json:"file"`
	Line      int       `json:"line"`
	Severity  Severity  `json:"severity"`
	Stack     string    `json:"stack"`
	Raw       string    `json:"raw"`
	Snippet   string    `json:"snippet,omitempty"`
	Context   []string  `json:"context,omitempty"`
	Samples   []string  `json:"samples,omitempty"`
	Hash      string    `json:"hash"`
	Count     int       `json:"count"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
}

var (
	uuidRe   = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	hexRe    = regexp.MustCompile(`(?i)\b[0-9a-f]{16,}\b`)
	quotedRe = regexp.MustCompile(`'[^']{1,240}'|"[^"]{1,240}"`)
	numRe    = regexp.MustCompile(`\b\d{3,}\b`)
)

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
	msg = uuidRe.ReplaceAllString(msg, "#id")
	msg = hexRe.ReplaceAllString(msg, "#hex")
	msg = quotedRe.ReplaceAllString(msg, "#str")
	msg = numRe.ReplaceAllString(msg, "#n")
	fields := strings.Fields(msg)
	return strings.Join(fields, " ")
}

// PushSample appends a distinct message, keeping the most recent few.
func PushSample(samples []string, msg string) []string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return samples
	}
	for _, s := range samples {
		if s == msg {
			return samples
		}
	}
	out := append(append([]string{}, samples...), msg)
	if len(out) > maxSamples {
		out = out[len(out)-maxSamples:]
	}
	return out
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
