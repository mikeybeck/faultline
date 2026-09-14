package source

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestCommandEmitsStdout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh -c")
	}
	c := &Command{Name: "dev", Type: "command", Path: "printf 'ERROR TypeError: boom in /tmp/x.js:1\\n'"}
	out := make(chan LineEvent, 4)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.Run(ctx, out); err != nil {
		t.Fatal(err)
	}
	select {
	case line := <-out:
		if line.Source != "dev" {
			t.Fatalf("source = %q", line.Source)
		}
		if line.Line != "ERROR TypeError: boom in /tmp/x.js:1" {
			t.Fatalf("line = %q", line.Line)
		}
	default:
		t.Fatal("no output")
	}
}
