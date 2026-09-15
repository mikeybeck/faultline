package sourcemap

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var sourceMapURLRe = regexp.MustCompile(`(?m)(?://[#@]|/\*)\s*sourceMappingURL=(\S+)`)

// Resolver looks up original locations from generated JS using nearby .map files.
type Resolver struct {
	mu      sync.Mutex
	maps    map[string]*Map
	miss    map[string]struct{}
	project string
	client  *http.Client
}

func NewResolver(projectDir string) *Resolver {
	return &Resolver{
		maps:    make(map[string]*Map),
		miss:    make(map[string]struct{}),
		project: projectDir,
		client:  &http.Client{Timeout: 400 * time.Millisecond},
	}
}

// Remap generated file/line/column onto original source when a map is found.
func (r *Resolver) Remap(file string, line, col int) (string, int, bool) {
	if r == nil || strings.TrimSpace(file) == "" || line <= 0 {
		return file, line, false
	}
	m := r.load(file)
	if m == nil {
		return file, line, false
	}
	orig, ol, _, ok := m.Lookup(line, col)
	if !ok {
		return file, line, false
	}
	if strings.Contains(orig, "://") {
		return orig, ol, true
	}
	if filepath.IsAbs(orig) {
		return orig, ol, true
	}
	base := filepath.Dir(localPath(file))
	if base != "" && base != "." {
		joined := filepath.Clean(filepath.Join(base, orig))
		if r.fileExists(joined) {
			return joined, ol, true
		}
	}
	if r.project != "" {
		cand := filepath.Join(r.project, orig)
		if r.fileExists(cand) {
			return cand, ol, true
		}
	}
	return orig, ol, true
}

func (r *Resolver) load(file string) *Map {
	key := strings.TrimSpace(file)
	r.mu.Lock()
	if m, ok := r.maps[key]; ok {
		r.mu.Unlock()
		return m
	}
	if _, miss := r.miss[key]; miss {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	m := r.fetch(file)
	r.mu.Lock()
	defer r.mu.Unlock()
	if m == nil {
		r.miss[key] = struct{}{}
		return nil
	}
	r.maps[key] = m
	return m
}

func (r *Resolver) fetch(file string) *Map {
	if data, ok := r.readBeside(file); ok {
		m, err := Parse(data)
		if err == nil {
			return m
		}
	}
	if data, urlOK := r.fetchHTTP(file); urlOK {
		m, err := Parse(data)
		if err == nil {
			return m
		}
	}
	return r.findByBase(file)
}

func (r *Resolver) readBeside(file string) ([]byte, bool) {
	p := localPath(file)
	if p == "" {
		return nil, false
	}
	if data, err := r.readFile(p + ".map"); err == nil {
		return data, true
	}
	js, err := r.readFile(p)
	if err != nil {
		return nil, false
	}
	if u := mappingURL(string(js)); u != "" && !strings.Contains(u, "://") {
		data, err := r.readFile(filepath.Join(filepath.Dir(p), u))
		if err == nil {
			return data, true
		}
	}
	return nil, false
}

func (r *Resolver) fetchHTTP(file string) ([]byte, bool) {
	jsURL := strings.TrimSpace(file)
	if !isLocalURL(jsURL) {
		return nil, false
	}
	body, err := r.get(jsURL)
	if err == nil {
		if ref := mappingURL(string(body)); ref != "" {
			mapURL := resolveURL(jsURL, ref)
			if data, err := r.get(mapURL); err == nil {
				return data, true
			}
		}
	}
	if data, err := r.get(jsURL + ".map"); err == nil {
		return data, true
	}
	return nil, false
}

func (r *Resolver) get(raw string) ([]byte, error) {
	if !isLocalURL(raw) {
		return nil, io.EOF
	}
	resp, err := r.client.Get(raw)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, io.EOF
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

func (r *Resolver) findByBase(file string) *Map {
	if r.project == "" {
		return nil
	}
	base := filepath.Base(localPath(file))
	if base == "" || base == "." {
		return nil
	}
	mapName := base + ".map"
	var hit string
	n := 0
	_ = filepath.WalkDir(r.project, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if path != r.project {
				name := d.Name()
				if name == "node_modules" || name == ".git" || name == "vendor" {
					return filepath.SkipDir
				}
			}
			return nil
		}
		n++
		if n > 6000 {
			return filepath.SkipAll
		}
		if d.Name() == mapName {
			hit = path
			return filepath.SkipAll
		}
		return nil
	})
	if hit == "" {
		return nil
	}
	data, err := r.readFile(hit)
	if err != nil {
		return nil
	}
	m, err := Parse(data)
	if err != nil {
		return nil
	}
	return m
}

func mappingURL(js string) string {
	// Prefer the last sourceMappingURL in the file.
	var last string
	sc := bufio.NewScanner(strings.NewReader(js))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		if m := sourceMapURLRe.FindStringSubmatch(sc.Text()); len(m) == 2 {
			last = strings.TrimRight(m[1], "*/")
		}
	}
	if last == "" {
		if m := sourceMapURLRe.FindStringSubmatch(js); len(m) == 2 {
			last = strings.TrimRight(m[1], "*/")
		}
	}
	if strings.HasPrefix(last, "data:") {
		return ""
	}
	return last
}

func resolveURL(base, ref string) string {
	if strings.Contains(ref, "://") {
		return ref
	}
	u, err := url.Parse(base)
	if err != nil {
		return ref
	}
	rel, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return u.ResolveReference(rel).String()
}

func localPath(file string) string {
	file = strings.TrimSpace(file)
	if file == "" {
		return ""
	}
	if u, err := url.Parse(file); err == nil && (u.Scheme == "http" || u.Scheme == "https" || u.Scheme == "file") {
		if u.Scheme == "file" {
			return u.Path
		}
		return u.Path
	}
	if i := strings.IndexAny(file, "?#"); i >= 0 {
		file = file[:i]
	}
	return file
}

func isLocalURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	return isLoopback(u.Hostname())
}

func isLoopback(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (r *Resolver) fileExists(path string) bool {
	if r == nil || r.project == "" || path == "" {
		return false
	}
	root := filepath.Clean(r.project)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if !filepath.IsLocal(rel) {
		return false
	}
	dir, err := os.OpenRoot(root)
	if err != nil {
		return false
	}
	defer dir.Close()
	_, err = dir.Stat(rel)
	return err == nil
}

func (r *Resolver) readFile(path string) ([]byte, error) {
	if r == nil || r.project == "" || path == "" {
		return nil, os.ErrNotExist
	}
	root := filepath.Clean(r.project)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, os.ErrNotExist
	}
	if !filepath.IsLocal(rel) {
		return nil, os.ErrNotExist
	}
	dir, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	f, err := dir.Open(rel)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
