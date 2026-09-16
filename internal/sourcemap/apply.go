package sourcemap

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mikey/faultline/internal/event"
)

var locRe = regexp.MustCompile(`((?:https?:\/\/|webpack-internal:\/\/\/|file:\/\/)?[^\s)]+\.(?:js|jsx|mjs|cjs|ts|tsx|vue|svelte))(?:\?[^:)\s]*)?:(\d+)(?::(\d+))?`)

// Apply rewrites ev.File/Line and stack frames using source maps when present.
func Apply(r *Resolver, ev *event.Event, column int) {
	if r == nil || ev == nil {
		return
	}
	if ev.File != "" && ev.Line > 0 {
		gen, genLine := ev.File, ev.Line
		if file, line, ok := r.Remap(ev.File, ev.Line, column); ok {
			ev.File = file
			ev.Line = line
		}
		if ev.Snippet == "" {
			ev.Snippet = r.Snippet(ev.File, ev.Line)
		}
		if ev.Snippet == "" {
			ev.Snippet = r.snippetFromGenerated(gen, genLine, column)
		}
	}
	if ev.Stack != "" {
		ev.Stack = remapStack(r, ev.Stack)
	}
	ev.Hash = event.Fingerprint(ev.Source, ev.Type, ev.Message, filepath.Base(ev.File), ev.Line)
}

func remapStack(r *Resolver, stack string) string {
	return locRe.ReplaceAllStringFunc(stack, func(m string) string {
		sub := locRe.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		file := sub[1]
		line, _ := strconv.Atoi(sub[2])
		col := 0
		if len(sub) >= 4 && sub[3] != "" {
			col, _ = strconv.Atoi(sub[3])
		}
		orig, ol, ok := r.Remap(file, line, col)
		if !ok {
			return m
		}
		old := file + ":" + sub[2]
		if col > 0 {
			old += ":" + sub[3]
		}
		next := orig + ":" + strconv.Itoa(ol)
		if col > 0 {
			next += ":" + sub[3]
		}
		return strings.Replace(m, old, next, 1)
	})
}
