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
	Enabled bool
	Sound   bool
}

// NewError sends a notification for a newly seen fingerprint.
func (n *Notifier) NewError(ev event.Event) {
	if !n.Enabled {
		return
	}
	title := fmt.Sprintf("New %s Exception", displaySource(ev.Source))
	body := ev.Title()
	if loc := ev.Location(); loc != "" {
		body = fmt.Sprintf("%s in %s", ev.Title(), loc)
	} else if ev.Message != "" {
		body = truncate(ev.Message, 120)
	}
	_ = send(title, body, n.Sound)
}

func displaySource(source string) string {
	if source == "" {
		return "Faultline"
	}
	// Title-case first letter.
	return strings.ToUpper(source[:1]) + source[1:]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func send(title, body string, sound bool) error {
	switch runtime.GOOS {
	case "linux":
		args := []string{"-a", "Faultline", title, body}
		if sound {
			args = append(args, "-h", "string:sound-name:dialog-warning")
		}
		return exec.Command("notify-send", args...).Run()
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, body, title)
		if sound {
			script = fmt.Sprintf(`display notification %q with title %q sound name "Glass"`, body, title)
		}
		return exec.Command("osascript", "-e", script).Run()
	case "windows":
		// Portable .exe: flash the taskbar button. Toasts need a registered
		// AppUserModelID (usually an installer) and are skipped on purpose.
		return flashTaskbar(sound)
	default:
		return nil
	}
}
