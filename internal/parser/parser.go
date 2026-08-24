package parser

import "github.com/mikey/faultline/internal/event"

// Parser converts raw log lines into structured events.
// Implementations may be stateful for multi-line records.
type Parser interface {
	// Feed ingests a single line. Completed events are returned
	// (may be empty). Flush may later return a pending event.
	Feed(line string) []*event.Event
	// Flush emits any pending incomplete event.
	Flush() []*event.Event
}

// ForType returns a parser for a configured source type.
func ForType(sourceType, sourceName string) (Parser, error) {
	switch sourceType {
	case "laravel":
		return NewLaravel(sourceName), nil
	case "apache":
		return NewApache(sourceName), nil
	default:
		return nil, errUnsupported(sourceType)
	}
}

type unsupportedError string

func (e unsupportedError) Error() string {
	return "unsupported parser type: " + string(e)
}

func errUnsupported(t string) error {
	return unsupportedError(t)
}
