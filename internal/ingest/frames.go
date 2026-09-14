package ingest

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mikey/faultline/internal/detect"
)

var (
	pythonFileRe = regexp.MustCompile(`File "([^"]+)", line (\d+)`)
	phpFrameRe   = regexp.MustCompile(`#\d+\s+.*?(?:[A-Za-z]:)?(/[^\s:(]+\.php)\((\d+)\)`)
	goFileRe     = regexp.MustCompile(`((?:[A-Za-z]:)?[^\s]+?\.go):(\d+)`)
)

// Frame is one stack line, optionally mapped onto a project file.
type Frame struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

// ParseFrames splits a stack blob into frames and resolves files when possible.
func ParseFrames(stack, projectDir string) []Frame {
	stack = strings.TrimRight(stack, "\n")
	if stack == "" {
		return nil
	}
	lines := strings.Split(stack, "\n")
	out := make([]Frame, 0, len(lines))
	for _, line := range lines {
		fr := Frame{Text: strings.TrimRight(line, "\r")}
		file, ln := locFromLine(fr.Text)
		if file != "" {
			fr.File = ResolveFile(file, projectDir)
			fr.Line = ln
		}
		out = append(out, fr)
	}
	return out
}

func locFromLine(line string) (string, int) {
	if m := stackLocRe.FindStringSubmatch(line); len(m) >= 3 {
		n, _ := strconv.Atoi(m[2])
		return m[1], n
	}
	if m := pythonFileRe.FindStringSubmatch(line); len(m) >= 3 {
		n, _ := strconv.Atoi(m[2])
		return m[1], n
	}
	if m := phpFrameRe.FindStringSubmatch(line); len(m) >= 3 {
		n, _ := strconv.Atoi(m[2])
		return m[1], n
	}
	if m := goFileRe.FindStringSubmatch(line); len(m) >= 3 {
		n, _ := strconv.Atoi(m[2])
		return m[1], n
	}
	return "", 0
}

func findByBase(projectDir, hint string) string {
	if projectDir == "" || hint == "" {
		return ""
	}
	base := filepath.Base(hint)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}
	hintSlash := strings.TrimPrefix(filepath.ToSlash(hint), "/")
	var suffixHits []string
	var baseHits []string
	n := 0
	_ = filepath.WalkDir(projectDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != projectDir && detect.ShouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		n++
		if n > 8000 {
			return fs.SkipAll
		}
		if d.Name() != base {
			return nil
		}
		rel, relErr := filepath.Rel(projectDir, path)
		if relErr != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		if hintSlash != "" && (strings.HasSuffix(hintSlash, relSlash) || strings.HasSuffix(relSlash, hintSlash) || strings.Contains(relSlash, hintSlash)) {
			suffixHits = append(suffixHits, path)
			return nil
		}
		baseHits = append(baseHits, path)
		return nil
	})
	if len(suffixHits) == 1 {
		return suffixHits[0]
	}
	if len(suffixHits) > 1 {
		return shortestPath(suffixHits)
	}
	if len(baseHits) == 1 {
		return baseHits[0]
	}
	return ""
}

func shortestPath(paths []string) string {
	best := paths[0]
	for _, p := range paths[1:] {
		if len(p) < len(best) {
			best = p
		}
	}
	return best
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
