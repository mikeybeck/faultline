package parser

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mikey/faultline/internal/event"
)

var (
	genericTSRe          = regexp.MustCompile(`^\[?(\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?)\]?\s*`)
	genericLevelRe       = regexp.MustCompile(`(?i)\b(emerg(?:ency)?|alert|crit(?:ical)?|fatal|panic|error|err|warn(?:ing)?|exception|traceback)\b`)
	genericTypeRe        = regexp.MustCompile(`(?i)\b([A-Za-z][\w.$]*?(?:Error|Exception|Throwable|Panic)|panic|IntegrityError|TypeError|ValueError|RuntimeError|ReferenceError|SyntaxError|AssertionError)\b`)
	genericPythonFileRe  = regexp.MustCompile(`File "([^"]+)", line (\d+)`)
	genericJSFileRe      = regexp.MustCompile(`\(?((?:[A-Za-z]:)?[^()\s]+?\.(?:js|jsx|ts|tsx|mjs|cjs)):(\d+)(?::\d+)?\)?`)
	genericGoFileRe      = regexp.MustCompile(`((?:[A-Za-z]:)?[^\s]+?\.go):(\d+)`)
	genericJavaFileRe    = regexp.MustCompile(`\(([\w$.]+\.java):(\d+)\)`)
	genericPHPFileRe     = regexp.MustCompile(`(?i)(?:in|at)\s+((?:[A-Za-z]:)?[^\s:]+?\.(?:php|phtml))\s*(?:on\s+line\s+|:)(\d+)`)
	genericRubyFileRe    = regexp.MustCompile(`from ((?:[A-Za-z]:)?[^\s:]+?\.rb):(\d+)`)
	genericGenericFileRe = regexp.MustCompile(`((?:[A-Za-z]:)?[^\s:]+?\.(?:py|js|jsx|ts|tsx|go|java|rb|php|rs|cs)):(\d+)`)
	genericStackFrameRe  = regexp.MustCompile(`^#\d+\s+`)
	genericNoiseRe       = regexp.MustCompile(`(?i)^\s*(info|debug|trace|notice)\b`)
)

// Generic parses arbitrary log files into error events.
type Generic struct {
	source  string
	current *pendingGeneric
}

type pendingGeneric struct {
	headerTime   time.Time
	typ          string
	message      string
	severity     event.Severity
	stackLines   []string
	raw          strings.Builder
	goDump       bool
	hasTraceback bool
}

func NewGeneric(source string) *Generic {
	return &Generic{source: source}
}

func (p *Generic) Feed(line string) []*event.Event {
	line = strings.TrimRight(line, "\r")
	trimmed := strings.TrimSpace(line)

	if jsonEv := ParseJSONLine(p.source, trimmed); jsonEv != nil {
		var out []*event.Event
		if p.current != nil {
			if done := p.finish(); done != nil {
				out = append(out, done)
			}
		}
		return append(out, jsonEv)
	}

	if p.current != nil {
		if trimmed == "" {
			// Keep the record open: Go panics and others put a blank before the stack.
			p.current.raw.WriteByte('\n')
			return nil
		}
		if isGenericContinuation(line, trimmed, p.current) || isPythonExceptionLine(trimmed, p.current) {
			p.appendLine(line, trimmed)
			if isPythonExceptionLine(trimmed, p.current) {
				applyGenericTypeMessage(p.current, trimmed)
			}
			return nil
		}
		var out []*event.Event
		if ev := p.finish(); ev != nil {
			out = append(out, ev)
		}
		if ev := p.start(line, trimmed); ev != nil {
			out = append(out, ev)
		}
		return out
	}

	if trimmed == "" {
		return nil
	}
	if ev := p.start(line, trimmed); ev != nil {
		return []*event.Event{ev}
	}
	return nil
}

func (p *Generic) Flush() []*event.Event {
	if p.current == nil {
		return nil
	}
	ev := p.finish()
	if ev == nil {
		return nil
	}
	return []*event.Event{ev}
}

func (p *Generic) start(line, trimmed string) *event.Event {
	if !looksLikeGenericError(trimmed) {
		return nil
	}
	sev := genericSeverity(trimmed)
	if sev == event.SeverityInfo {
		return nil
	}
	lower := strings.ToLower(trimmed)
	p.current = &pendingGeneric{
		headerTime:   parseGenericTime(trimmed),
		severity:     sev,
		goDump:       strings.Contains(lower, "panic") || strings.Contains(lower, "goroutine "),
		hasTraceback: strings.Contains(lower, "traceback"),
	}
	p.current.raw.WriteString(line)
	applyGenericTypeMessage(p.current, stripGenericPrefix(trimmed))
	return nil
}

func (p *Generic) appendLine(line, trimmed string) {
	if p.current == nil {
		return
	}
	p.current.raw.WriteByte('\n')
	p.current.raw.WriteString(line)
	p.current.stackLines = append(p.current.stackLines, line)
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "panic") || strings.Contains(lower, "goroutine ") {
		p.current.goDump = true
	}
	if strings.Contains(lower, "traceback") {
		p.current.hasTraceback = true
	}
}

func (p *Generic) finish() *event.Event {
	cur := p.current
	p.current = nil
	if cur == nil {
		return nil
	}
	if cur.severity == event.SeverityInfo {
		return nil
	}

	body := cur.message
	if body == "" {
		body = strings.TrimSpace(cur.raw.String())
	}
	file, ln := extractGenericFileLine(cur.raw.String())
	typ := cur.typ
	if typ == "" {
		typ = "Error"
	}
	now := cur.headerTime
	if now.IsZero() {
		now = time.Now()
	}
	hash := event.Fingerprint(p.source, typ, body, filepath.Base(file), ln)
	return &event.Event{
		Source:    p.source,
		Time:      now,
		Type:      typ,
		Message:   body,
		File:      file,
		Line:      ln,
		Severity:  cur.severity,
		Stack:     strings.TrimSpace(strings.Join(cur.stackLines, "\n")),
		Raw:       cur.raw.String(),
		Hash:      hash,
		Count:     1,
		FirstSeen: now,
		LastSeen:  now,
	}
}

func looksLikeGenericError(line string) bool {
	lower := strings.ToLower(line)
	if genericNoiseRe.MatchString(line) && !genericLevelRe.MatchString(line) {
		return false
	}
	keys := []string{
		"error", "exception", "fatal", "panic", "traceback", "failed", "failure",
		"warning", "warn", "critical", "emergency", "segfault", "uncaught",
		"unhandled", "denied",
	}
	for _, k := range keys {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return genericTypeRe.MatchString(line)
}

func genericSeverity(line string) event.Severity {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "fatal"), strings.Contains(lower, "panic"),
		strings.Contains(lower, "emerg"), strings.Contains(lower, "critical"),
		strings.Contains(lower, "alert"):
		return event.SeverityCritical
	case strings.Contains(lower, "warn"), strings.Contains(lower, "deprecated"):
		return event.SeverityWarning
	case strings.Contains(lower, "error"), strings.Contains(lower, "exception"),
		strings.Contains(lower, "traceback"), strings.Contains(lower, "fail"),
		strings.Contains(lower, "uncaught"), strings.Contains(lower, "unhandled"):
		return event.SeverityError
	default:
		if genericTypeRe.MatchString(line) {
			return event.SeverityError
		}
		return event.SeverityInfo
	}
}

func isGenericContinuation(line, trimmed string, cur *pendingGeneric) bool {
	if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		return true
	}
	lower := strings.ToLower(trimmed)
	switch {
	case strings.HasPrefix(trimmed, "at "), strings.HasPrefix(trimmed, "at\t"):
		return true
	case strings.HasPrefix(trimmed, "File "), strings.HasPrefix(trimmed, `File "`):
		return true
	case genericStackFrameRe.MatchString(trimmed):
		return true
	case strings.HasPrefix(lower, "from ") && strings.Contains(lower, ".rb"):
		return true
	case strings.HasPrefix(lower, "goroutine "):
		return true
	case strings.HasPrefix(lower, "caused by:"):
		return true
	case strings.HasPrefix(lower, "[stacktrace]"), trimmed == "Stack trace:":
		return true
	case isGoDumpContinuation(trimmed, cur):
		return true
	default:
		return false
	}
}

func isGoDumpContinuation(trimmed string, cur *pendingGeneric) bool {
	if cur == nil || !cur.goDump {
		return false
	}
	if strings.HasPrefix(trimmed, "created by ") {
		return true
	}
	if strings.Contains(trimmed, "(") && strings.Contains(trimmed, ")") && !looksLikeGenericError(trimmed) {
		return true
	}
	return false
}

func isPythonExceptionLine(trimmed string, cur *pendingGeneric) bool {
	if cur == nil || !cur.hasTraceback {
		return false
	}
	if cur.typ != "" && !strings.EqualFold(cur.typ, "Traceback") {
		return false
	}
	if strings.HasPrefix(trimmed, " ") || strings.HasPrefix(trimmed, "\t") || strings.HasPrefix(trimmed, "File ") {
		return false
	}
	m := genericTypeRe.FindStringSubmatch(trimmed)
	if m == nil {
		return false
	}
	return strings.HasPrefix(trimmed, m[1])
}

func applyGenericTypeMessage(cur *pendingGeneric, line string) {
	line = strings.TrimSpace(line)
	if m := genericTypeRe.FindStringSubmatch(line); m != nil {
		cur.typ = m[1]
		if i := strings.Index(line, m[1]); i >= 0 {
			rest := strings.TrimSpace(line[i+len(m[1]):])
			rest = strings.TrimPrefix(rest, ":")
			rest = strings.TrimSpace(rest)
			if rest != "" {
				cur.message = rest
			} else {
				cur.message = line
			}
		}
		return
	}
	if cur.typ == "" {
		if strings.HasPrefix(strings.ToLower(line), "traceback") {
			cur.typ = "Traceback"
			cur.message = line
			return
		}
		if idx := strings.Index(line, ":"); idx > 0 && idx < 80 {
			head := strings.TrimSpace(line[:idx])
			if !strings.Contains(head, " ") {
				cur.typ = head
				cur.message = strings.TrimSpace(line[idx+1:])
				return
			}
		}
		cur.typ = "Error"
		cur.message = line
	}
}

func stripGenericPrefix(line string) string {
	line = genericTSRe.ReplaceAllString(line, "")
	line = strings.TrimSpace(line)
	return line
}

func parseGenericTime(line string) time.Time {
	if m := genericTSRe.FindStringSubmatch(line); m != nil {
		s := strings.ReplaceAll(m[1], "T", " ")
		for _, f := range []string{
			"2006-01-02 15:04:05",
			"2006-01-02 15:04:05.000000",
			"2006-01-02 15:04:05.000",
		} {
			if t, err := time.ParseInLocation(f, s, time.Local); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func extractGenericFileLine(text string) (string, int) {
	type hit struct {
		file string
		line int
		rank int
	}
	var hits []hit
	collect := func(re *regexp.Regexp, rank int) {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			if len(m) < 3 {
				continue
			}
			ln, _ := strconv.Atoi(m[2])
			hits = append(hits, hit{file: m[1], line: ln, rank: rank})
		}
	}
	collect(genericPythonFileRe, 3)
	collect(genericJSFileRe, 3)
	collect(genericGoFileRe, 3)
	collect(genericJavaFileRe, 3)
	collect(genericPHPFileRe, 3)
	collect(genericRubyFileRe, 3)
	collect(genericGenericFileRe, 1)

	var best hit
	for _, h := range hits {
		if isNoisyFrame(h.file) {
			h.rank -= 2
		}
		if best.file == "" || h.rank > best.rank {
			best = h
		}
	}
	return best.file, best.line
}

func isNoisyFrame(file string) bool {
	norm := filepath.ToSlash(strings.ToLower(file))
	for _, p := range []string{
		"/node_modules/", "node_modules/",
		"/vendor/", "vendor/",
		"/site-packages/",
		"internal/modules/",
		"node:internal/",
	} {
		if strings.Contains(norm, p) {
			return true
		}
	}
	return false
}
