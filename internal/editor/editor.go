package editor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
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
	return cmd.Start()
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
		if line > 0 {
			return bin, []string{"--goto", fmt.Sprintf("%s:%d", file, line)}, nil
		}
		return bin, []string{file}, nil
	case "phpstorm", "idea":
		bin := resolvePhpStorm()
		if line > 0 {
			return bin, []string{"--line", strconv.Itoa(line), file}, nil
		}
		return bin, []string{file}, nil
	}

	// Custom template with {file} and {line}.
	if strings.Contains(spec, "{file}") {
		replaced := strings.ReplaceAll(spec, "{file}", shellQuote(file))
		replaced = strings.ReplaceAll(replaced, "{line}", strconv.Itoa(max(line, 1)))
		return "sh", []string{"-c", replaced}, nil
	}

	// Treat as binary name.
	if line > 0 {
		return spec, []string{fmt.Sprintf("+%d", line), file}, nil
	}
	return spec, []string{file}, nil
}

func resolvePhpStorm() string {
	candidates := []string{"phpstorm", "phpstorm.sh", "pstorm"}
	if runtime.GOOS == "darwin" {
		candidates = append([]string{"open"}, candidates...)
	}
	for _, c := range candidates {
		if c == "open" {
			continue
		}
		if path, err := exec.LookPath(c); err == nil {
			return path
		}
	}
	return "phpstorm"
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
