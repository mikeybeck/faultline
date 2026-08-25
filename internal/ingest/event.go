package ingest

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mikey/faultline/internal/event"
)

// Payload is the JSON body posted by the browser extension.
type Payload struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Stack    string `json:"stack"`
	URL      string `json:"url"`
	Severity string `json:"severity"`
}

var (
	stackLocRe = regexp.MustCompile(`((?:https?:\/\/|webpack-internal:\/\/\/|file:\/\/)[^\s)]+?\.(?:js|jsx|mjs|cjs|ts|tsx|vue|svelte|php)|[^\s:)]+\.(?:js|jsx|mjs|cjs|ts|tsx|vue|svelte|php)):(\d+)(?::(\d+))?`)
)

// ToEvent converts a browser payload into a store event.
func ToEvent(sourceName, projectDir string, p Payload) event.Event {
	if sourceName == "" {
		sourceName = DefaultName
	}
	typ := strings.TrimSpace(p.Type)
	if typ == "" {
		typ = "Error"
	}
	msg := strings.TrimSpace(p.Message)
	if msg == "" {
		msg = typ
	}
	file := strings.TrimSpace(p.File)
	line := p.Line
	if file == "" || line <= 0 {
		f, l := fileLineFromStack(p.Stack)
		if file == "" {
			file = f
		}
		if line <= 0 {
			line = l
		}
	}
	file = ResolveFile(file, projectDir)
	now := time.Now()
	rawParts := []string{typ + ": " + msg}
	if u := strings.TrimSpace(p.URL); u != "" {
		rawParts = append(rawParts, u)
	}
	if st := strings.TrimSpace(p.Stack); st != "" {
		rawParts = append(rawParts, st)
	}
	raw := strings.Join(rawParts, "\n")
	return event.Event{
		Source:    sourceName,
		Time:      now,
		Type:      typ,
		Message:   msg,
		File:      file,
		Line:      line,
		Severity:  parseSeverity(p.Severity),
		Stack:     strings.TrimSpace(p.Stack),
		Raw:       raw,
		Hash:      event.Fingerprint(sourceName, typ, msg, filepath.Base(file), line),
		Count:     1,
		FirstSeen: now,
		LastSeen:  now,
	}
}

func parseSeverity(s string) event.Severity {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical", "fatal":
		return event.SeverityCritical
	case "warning", "warn":
		return event.SeverityWarning
	case "info":
		return event.SeverityInfo
	default:
		return event.SeverityError
	}
}

func fileLineFromStack(stack string) (string, int) {
	if stack == "" {
		return "", 0
	}
	var noisyFile string
	var noisyLine int
	for _, m := range stackLocRe.FindAllStringSubmatch(stack, -1) {
		file := m[1]
		line, _ := strconv.Atoi(m[2])
		if isNoisyBrowserFrame(file) {
			if noisyFile == "" {
				noisyFile, noisyLine = file, line
			}
			continue
		}
		return file, line
	}
	return noisyFile, noisyLine
}

// ResolveFile turns a browser filename (often a URL) into a project path when possible.
func ResolveFile(file, projectDir string) string {
	file = strings.TrimSpace(file)
	if file == "" {
		return ""
	}
	file = stripURL(file)
	file = strings.ReplaceAll(file, "\\", "/")
	if strings.HasPrefix(file, "/@fs/") {
		file = strings.TrimPrefix(file, "/@fs")
	} else if strings.HasPrefix(file, "@fs/") {
		file = strings.TrimPrefix(file, "@fs")
	}
	if projectDir == "" {
		return strings.TrimPrefix(file, "/")
	}
	if filepath.IsAbs(file) {
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}
	rel := strings.TrimPrefix(file, "/")
	cand := filepath.Join(projectDir, rel)
	if _, err := os.Stat(cand); err == nil {
		return cand
	}
	return rel
}

func stripURL(file string) string {
	if strings.HasPrefix(file, "webpack-internal:///") {
		return strings.TrimPrefix(file, "webpack-internal:///")
	}
	u, err := url.Parse(file)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "file") {
		if i := strings.Index(file, "?"); i >= 0 {
			file = file[:i]
		}
		if i := strings.Index(file, "#"); i >= 0 {
			file = file[:i]
		}
		return file
	}
	if u.Scheme == "file" {
		return u.Path
	}
	return u.Path
}

func isNoisyBrowserFrame(file string) bool {
	norm := strings.ToLower(file)
	for _, p := range []string{
		"/node_modules/", "node_modules/",
		"chrome-extension://", "moz-extension://",
		"webpack/bootstrap",
		"vite/client",
		"/@vite/",
	} {
		if strings.Contains(norm, p) {
			return true
		}
	}
	return false
}
