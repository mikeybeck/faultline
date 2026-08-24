package source

import "time"

// State describes health of a watched source.
type State string

const (
	StateWaiting State = "waiting" // file missing, waiting for create
	StateOK      State = "ok"
	StateError   State = "error"
)

// Status is reported to the TUI for each source.
type Status struct {
	Name    string
	Type    string
	Path    string
	State   State
	Message string
	Updated time.Time
}
