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
	Enabled      bool
	Sound        bool
	ActivateBase string
	OnActivate   func(event.Event)
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
	clickURL := ""
	if n.ActivateBase != "" && ev.Hash != "" {
		clickURL = strings.TrimRight(n.ActivateBase, "/") + "/activate?hash=" + ev.Hash
	}
	go func() {
		_ = send(title, body, n.Sound, clickURL, func() {
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

func send(title, body string, sound bool, clickURL string, onClick func()) error {
	switch runtime.GOOS {
	case "linux":
		return sendLinux(title, body, sound, onClick)
	case "darwin":
		return sendDarwin(title, body, sound, clickURL, onClick)
	case "windows":
		return sendWindows(title, body, sound, onClick)
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

func sendDarwin(title, body string, sound bool, clickURL string, onClick func()) error {
	if path, err := exec.LookPath("terminal-notifier"); err == nil {
		args := []string{"-title", title, "-message", body, "-timeout", "12", "-group", "faultline"}
		if sound {
			args = append(args, "-sound", "default")
		}
		if clickURL != "" {
			args = append(args, "-execute", "curl -fsS "+shellQuote(clickURL))
		}
		if err := exec.Command(path, args...).Start(); err == nil {
			return nil
		}
	}
	script := fmt.Sprintf(`display notification %q with title %q`, body, title)
	if sound {
		script = fmt.Sprintf(`display notification %q with title %q sound name "Glass"`, body, title)
	}
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		return err
	}
	_ = onClick
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
