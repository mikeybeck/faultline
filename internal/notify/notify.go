package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mikey/faultline/internal/event"
)

// Notifier sends desktop notifications for new errors.
type Notifier struct {
	Enabled    bool
	Sound      bool
	OnActivate func(event.Event)
}

// NewError sends a notification for a newly seen fingerprint.
func (n *Notifier) NewError(ev event.Event) {
	if n == nil || !n.Enabled {
		return
	}
	title := fmt.Sprintf("New %s Exception", displaySource(ev.Source))
	body := ev.Title()
	if loc := ev.Location(); loc != "" {
		body = fmt.Sprintf("%s in %s", ev.Title(), loc)
	} else if ev.Message != "" {
		body = truncate(ev.Message, 120)
	}
	click := n.OnActivate
	go func() {
		_ = send(title, body, n.Sound, func() {
			if click != nil {
				click(ev)
			}
		})
	}()
}

func displaySource(source string) string {
	if source == "" {
		return "Faultline"
	}
	return strings.ToUpper(source[:1]) + source[1:]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func send(title, body string, sound bool, onClick func()) error {
	switch runtime.GOOS {
	case "linux":
		return sendLinux(title, body, sound, onClick)
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, body, title)
		if sound {
			script = fmt.Sprintf(`display notification %q with title %q sound name "Glass"`, body, title)
		}
		return exec.Command("osascript", "-e", script).Run()
	case "windows":
		return flashTaskbar(sound)
	default:
		return nil
	}
}

func sendLinux(title, body string, sound bool, onClick func()) error {
	args := []string{"-a", "Faultline", title, body}
	if onClick != nil {
		args = []string{"-a", "Faultline", "--action=default=Open", "--wait", title, body}
	}
	if sound {
		args = append(args, "-h", "string:sound-name:dialog-warning")
	}
	if onClick == nil {
		return exec.Command("notify-send", args...).Run()
	}
	cmd := exec.Command("notify-send", args...)
	out, err := cmd.Output()
	if err == nil {
		if strings.TrimSpace(string(out)) == "default" {
			onClick()
		}
		return nil
	}
	fallback := []string{"-a", "Faultline", title, body}
	if sound {
		fallback = append(fallback, "-h", "string:sound-name:dialog-warning")
	}
	return exec.Command("notify-send", fallback...).Run()
}
