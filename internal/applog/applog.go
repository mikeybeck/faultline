package applog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const Source = "faultline"

var mu sync.Mutex

// Path is ~/.config/faultline/faultline.log (or the OS equivalent).
func Path() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("faultline log: %w", err)
	}
	return filepath.Join(base, "faultline", "faultline.log"), nil
}

// Write appends an error to the Faultline log file.
func Write(message, stack string) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("faultline log: %w", err)
	}
	message = strings.TrimSpace(message)
	if message == "" {
		message = "unknown error"
	}
	var b strings.Builder
	b.WriteString(time.Now().UTC().Format(time.RFC3339))
	b.WriteString(" ERROR ")
	b.WriteString(message)
	b.WriteByte('\n')
	if s := strings.TrimSpace(stack); s != "" {
		b.WriteString(s)
		if !strings.HasSuffix(s, "\n") {
			b.WriteByte('\n')
		}
	}
	b.WriteByte('\n')

	mu.Lock()
	defer mu.Unlock()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("faultline log: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(b.String()); err != nil {
		return fmt.Errorf("faultline log: %w", err)
	}
	return nil
}
