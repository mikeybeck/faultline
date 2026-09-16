package config

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mikey/faultline/internal/event"
)

// IgnoreRule hides matching events. Empty fields are wildcards; a rule matches
// when every set field matches (AND). Any matching rule ignores the event (OR).
type IgnoreRule struct {
	Type    string `yaml:"type,omitempty" json:"type,omitempty"`
	Message string `yaml:"message,omitempty" json:"message,omitempty"`
	Path    string `yaml:"path,omitempty" json:"path,omitempty"`
	Source  string `yaml:"source,omitempty" json:"source,omitempty"`
	Regex   string `yaml:"regex,omitempty" json:"regex,omitempty"`
}

func (r IgnoreRule) empty() bool {
	return strings.TrimSpace(r.Type) == "" &&
		strings.TrimSpace(r.Message) == "" &&
		strings.TrimSpace(r.Path) == "" &&
		strings.TrimSpace(r.Source) == "" &&
		strings.TrimSpace(r.Regex) == ""
}

func (r IgnoreRule) matches(ev event.Event) bool {
	if r.empty() {
		return false
	}
	if t := strings.TrimSpace(r.Type); t != "" && !strings.EqualFold(ev.Type, t) {
		return false
	}
	if src := strings.TrimSpace(r.Source); src != "" && ev.Source != src {
		return false
	}
	if msg := strings.TrimSpace(r.Message); msg != "" && !strings.Contains(strings.ToLower(ev.Message), strings.ToLower(msg)) {
		return false
	}
	if p := strings.TrimSpace(r.Path); p != "" && !matchPath(p, ev.File) {
		return false
	}
	if pat := strings.TrimSpace(r.Regex); pat != "" {
		re, err := regexp.Compile("(?i)" + pat)
		if err != nil {
			return false
		}
		if !re.MatchString(ev.Message) && !re.MatchString(ev.Type) && !re.MatchString(ev.File) {
			return false
		}
	}
	return true
}

// Ignores reports whether ev matches a checked-in ignore rule.
func (c *Config) Ignores(ev event.Event) bool {
	if c == nil {
		return false
	}
	for _, r := range c.Inbox.Ignore {
		if r.matches(ev) {
			return true
		}
	}
	return false
}

func validateIgnore(rules []IgnoreRule) error {
	for i, r := range rules {
		if r.empty() {
			return fmt.Errorf("config: inbox.ignore[%d] needs type, message, path, source, or regex", i)
		}
		if pat := strings.TrimSpace(r.Regex); pat != "" {
			if _, err := regexp.Compile("(?i)" + pat); err != nil {
				return fmt.Errorf("config: inbox.ignore[%d].regex: %w", i, err)
			}
		}
	}
	return nil
}

func matchPath(pattern, file string) bool {
	if file == "" {
		return false
	}
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	file = filepath.ToSlash(file)
	if pattern == "" {
		return false
	}
	if !strings.ContainsAny(pattern, "*?[") {
		return strings.Contains(file, pattern)
	}
	if ok, _ := path.Match(pattern, file); ok {
		return true
	}
	if ok, _ := path.Match(pattern, path.Base(file)); ok {
		return true
	}
	if strings.HasPrefix(pattern, "**/") {
		rest := strings.TrimPrefix(pattern, "**/")
		if matchPath(rest, file) || matchPath(rest, path.Base(file)) {
			return true
		}
		for i := 0; i < len(file); i++ {
			if file[i] == '/' && matchPath(rest, file[i+1:]) {
				return true
			}
		}
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return prefix != "" && (strings.HasPrefix(file, prefix+"/") || strings.Contains(file, "/"+prefix+"/"))
	}
	return false
}
