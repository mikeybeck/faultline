package source

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// Command runs a shell command and emits stdout/stderr lines as a log source.
type Command struct {
	Name     string
	Type     string
	Path     string // the command line
	Dir      string
	OnStatus StatusFunc
}

// Run starts the command and copies output until it exits or ctx is cancelled.
func (c *Command) Run(ctx context.Context, out chan<- LineEvent) error {
	report := func(state State, msg string) {
		if c.OnStatus == nil {
			return
		}
		c.OnStatus(Status{
			Name:    c.Name,
			Type:    c.Type,
			Path:    c.Path,
			State:   state,
			Message: msg,
			Updated: time.Now(),
		})
	}

	cmdline := c.Path
	if cmdline == "" {
		report(StateError, "empty command")
		return fmt.Errorf("empty command")
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", cmdline)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", cmdline)
	}
	cmd.Dir = c.Dir
	hideConsole(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		report(StateError, err.Error())
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		report(StateError, err.Error())
		return err
	}

	if err := cmd.Start(); err != nil {
		report(StateError, err.Error())
		return err
	}
	report(StateOK, "running")

	emit := func(r io.Reader) {
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			select {
			case <-ctx.Done():
				return
			case out <- LineEvent{Source: c.Name, Line: sc.Text(), Time: time.Now()}:
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); emit(stdout) }()
	go func() { defer wg.Done(); emit(stderr) }()

	waitErr := cmd.Wait()
	wg.Wait()

	if ctx.Err() != nil {
		return ctx.Err()
	}
	if waitErr != nil {
		report(StateError, waitErr.Error())
		return waitErr
	}
	report(StateWaiting, "exited")
	return nil
}
