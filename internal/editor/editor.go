package editor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Opener opens a file at a given line in the configured editor.
type Opener struct {
	Command string
}

// Open launches the editor for file:line.
func (o Opener) Open(file string, line int) error {
	if file == "" {
		return fmt.Errorf("no file associated with this error")
	}
	cmdName, args, err := o.build(file, line)
	if err != nil {
		return err
	}
	cmd := exec.Command(cmdName, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("%w — pick the editor executable in Settings (Browse)", err)
		}
		return err
	}
	return nil
}

func (o Opener) build(file string, line int) (string, []string, error) {
	spec := strings.TrimSpace(o.Command)
	if spec == "" {
		spec = "code"
	}

	switch strings.ToLower(spec) {
	case "code", "vscode", "cursor":
		bin := "code"
		if strings.EqualFold(spec, "cursor") {
			bin = "cursor"
		}
		return vscodeArgs(bin, file, line)
	case "phpstorm", "idea":
		return phpstormArgs(resolvePhpStorm(), file, line)
	}

	if strings.Contains(spec, "{file}") {
		return expandTemplate(spec, file, line)
	}

	bin := unquote(spec)
	switch flavorOf(bin) {
	case "phpstorm":
		return phpstormArgs(bin, file, line)
	case "vscode":
		return vscodeArgs(bin, file, line)
	}

	if line > 0 {
		return bin, []string{fmt.Sprintf("+%d", line), file}, nil
	}
	return bin, []string{file}, nil
}

func vscodeArgs(bin, file string, line int) (string, []string, error) {
	if line > 0 {
		return bin, []string{"--goto", fmt.Sprintf("%s:%d", file, line)}, nil
	}
	return bin, []string{file}, nil
}

func phpstormArgs(bin, file string, line int) (string, []string, error) {
	if line > 0 {
		return bin, []string{"--line", strconv.Itoa(line), file}, nil
	}
	return bin, []string{file}, nil
}

func resolvePhpStorm() string {
	names := []string{"phpstorm", "phpstorm.sh", "pstorm"}
	if runtime.GOOS == "windows" {
		names = []string{"phpstorm64.exe", "phpstorm.exe", "phpstorm.cmd", "phpstorm", "pstorm.cmd", "pstorm"}
	}
	for _, c := range names {
		if path, err := exec.LookPath(c); err == nil {
			return path
		}
	}
	if p := findPhpStormInstall(); p != "" {
		return p
	}
	if runtime.GOOS == "windows" {
		return "phpstorm64.exe"
	}
	return "phpstorm"
}

func findPhpStormInstall() string {
	for _, pattern := range phpStormGlobs() {
		if !strings.ContainsAny(pattern, "*?[") {
			if st, err := os.Stat(pattern); err == nil && !st.IsDir() {
				return pattern
			}
			continue
		}
		if p := newestGlob(pattern); p != "" {
			return p
		}
	}
	return ""
}

func phpStormGlobs() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		local := os.Getenv("LOCALAPPDATA")
		pf := os.Getenv("ProgramFiles")
		pf86 := os.Getenv("ProgramFiles(x86)")
		var globs []string
		if local != "" {
			globs = append(globs,
				filepath.Join(local, "JetBrains", "Toolbox", "scripts", "phpstorm.cmd"),
				filepath.Join(local, "JetBrains", "Toolbox", "scripts", "phpstorm.exe"),
				filepath.Join(local, "JetBrains", "Toolbox", "apps", "PhpStorm", "*", "*", "bin", "phpstorm64.exe"),
				filepath.Join(local, "Programs", "PhpStorm*", "bin", "phpstorm64.exe"),
			)
		}
		if pf != "" {
			globs = append(globs, filepath.Join(pf, "JetBrains", "PhpStorm*", "bin", "phpstorm64.exe"))
		}
		if pf86 != "" {
			globs = append(globs, filepath.Join(pf86, "JetBrains", "PhpStorm*", "bin", "phpstorm64.exe"))
		}
		return globs
	case "darwin":
		return []string{
			"/Applications/PhpStorm.app/Contents/MacOS/phpstorm",
			filepath.Join(home, "Applications", "PhpStorm.app", "Contents", "MacOS", "phpstorm"),
			filepath.Join(home, "Library", "Application Support", "JetBrains", "Toolbox", "scripts", "phpstorm"),
		}
	default:
		return []string{
			filepath.Join(home, ".local", "share", "JetBrains", "Toolbox", "scripts", "phpstorm"),
			filepath.Join(home, ".local", "share", "JetBrains", "Toolbox", "apps", "PhpStorm", "*", "*", "bin", "phpstorm.sh"),
			"/opt/phpstorm/bin/phpstorm.sh",
			"/opt/PhpStorm*/bin/phpstorm.sh",
		}
	}
}

func newestGlob(pattern string) string {
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}
	sort.Strings(matches)
	for i := len(matches) - 1; i >= 0; i-- {
		if st, err := os.Stat(matches[i]); err == nil && !st.IsDir() {
			return matches[i]
		}
	}
	return ""
}

func flavorOf(bin string) string {
	base := strings.ToLower(filepath.Base(bin))
	for _, ext := range []string{".exe", ".cmd", ".bat", ".sh"} {
		base = strings.TrimSuffix(base, ext)
	}
	switch {
	case strings.Contains(base, "phpstorm"), strings.Contains(base, "pstorm"), strings.HasPrefix(base, "idea"):
		return "phpstorm"
	case base == "code", strings.HasPrefix(base, "code-"), strings.Contains(base, "cursor"):
		return "vscode"
	default:
		return "generic"
	}
}

func expandTemplate(spec, file string, line int) (string, []string, error) {
	parts, err := splitCommand(spec)
	if err != nil {
		return "", nil, err
	}
	lineStr := strconv.Itoa(max(line, 1))
	for i, p := range parts {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(p, "{file}", file), "{line}", lineStr)
	}
	return parts[0], parts[1:], nil
}

func splitCommand(s string) ([]string, error) {
	var parts []string
	var buf strings.Builder
	quote := rune(0)
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				buf.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case unicode.IsSpace(r):
			if buf.Len() > 0 {
				parts = append(parts, buf.String())
				buf.Reset()
			}
		default:
			buf.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote in editor command")
	}
	if buf.Len() > 0 {
		parts = append(parts, buf.String())
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty editor command")
	}
	return parts, nil
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Installed returns editor ids found on this machine, preferred first.
func Installed() []string {
	var out []string
	seen := map[string]bool{}
	add := func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	if _, err := exec.LookPath("cursor"); err == nil {
		add("cursor")
	}
	if _, err := exec.LookPath("code"); err == nil {
		add("code")
	}
	if _, err := exec.LookPath("phpstorm"); err == nil {
		add("phpstorm")
	} else if _, err := exec.LookPath("phpstorm64.exe"); err == nil {
		add("phpstorm")
	} else if findPhpStormInstall() != "" {
		add("phpstorm")
	}
	return out
}

// PreferredCommand is the first installed editor, or "code".
func PreferredCommand() string {
	if inst := Installed(); len(inst) > 0 {
		return inst[0]
	}
	return "code"
}
