package detect

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	TypeGeneric = "generic"
	TypeLaravel = "laravel"
	TypeApache  = "apache"
)

var (
	laravelSniffRe = regexp.MustCompile(`(?m)^\[\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?\]\s+\w+\.\w+:`)
	apacheSniffRe  = regexp.MustCompile(`(?m)^\[(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun)\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}`)
)

var skipDirs = map[string]struct{}{
	"node_modules": {},
	".git":         {},
	"vendor":       {},
	"dist":         {},
	".next":        {},
	"target":       {},
	"__pycache__":  {},
	".venv":        {},
	"venv":         {},
	"build":        {},
	"coverage":     {},
	".tox":         {},
	".cache":       {},
}

const (
	maxDepth      = 5
	maxFiles      = 40
	sniffMaxBytes = 32 * 1024
)

// Candidate is a log file found in a project tree.
type Candidate struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

// Sniff guesses a parser type from a file's contents.
func Sniff(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return TypeGeneric
	}
	defer f.Close()
	buf := make([]byte, sniffMaxBytes)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return TypeGeneric
	}
	return SniffBytes(buf[:n])
}

// SniffBytes guesses a parser type from a sample.
func SniffBytes(data []byte) string {
	if laravelSniffRe.Match(data) {
		return TypeLaravel
	}
	if apacheSniffRe.Match(data) {
		return TypeApache
	}
	return TypeGeneric
}

// ScanDir finds likely log files under root and sniffs each format.
func ScanDir(root string) ([]Candidate, error) {
	root = filepath.Clean(root)
	var out []Candidate
	usedNames := map[string]int{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		depth := 0
		if rel != "." {
			depth = 1 + strings.Count(rel, string(os.PathSeparator))
		}
		if d.IsDir() {
			if path != root {
				if _, skip := skipDirs[d.Name()]; skip {
					return filepath.SkipDir
				}
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
			}
			if depth > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if len(out) >= maxFiles {
			return fs.SkipAll
		}
		if !isLogFile(d.Name()) {
			return nil
		}
		name := nameFromPath(d.Name(), usedNames)
		out = append(out, Candidate{
			Name: name,
			Path: path,
			Type: Sniff(path),
		})
		return nil
	})
	if err != nil {
		return out, err
	}
	return out, nil
}

func isLogFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".log") || strings.HasSuffix(lower, ".log.txt")
}

func nameFromPath(filename string, used map[string]int) string {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	base = strings.TrimSpace(base)
	if base == "" {
		base = "log"
	}
	n := used[base]
	used[base] = n + 1
	if n == 0 {
		return base
	}
	return base + "-" + itoa(n+1)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
