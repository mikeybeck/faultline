//go:build !windows

package notify

func flashTaskbar(sound bool) error {
	return nil
}

func sendWindows(title, body string, sound bool, onClick func()) error {
	_ = title
	_ = body
	_ = sound
	_ = onClick
	return nil
}
