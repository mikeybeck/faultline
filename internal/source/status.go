package source

import "time"

// State describes health of a watched source.
type State string

const (
	StateWaiting   State = "waiting"   // file missing, waiting for create
	StateIngesting State = "ingesting" // reading existing content from start
	StateOK        State = "ok"
	StateError     State = "error"
)

// Status is reported to the TUI for each source.
type Status struct {
	Name    string    `json:"name"`
	Type    string    `json:"type"`
	Path    string    `json:"path"`
	State   State     `json:"state"`
	Message string    `json:"message"`
	Updated time.Time `json:"updated"`
}
