//go:build !windows

package notify

func flashTaskbar(sound bool) error {
	return nil
}
