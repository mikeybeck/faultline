package source

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	pollInterval   = 200 * time.Millisecond
	reopenInterval = time.Second
)

// LineEvent is a raw line read from a followed file.
type LineEvent struct {
	Source string
	Line   string
	Time   time.Time
}

// StatusFunc receives source health updates.
type StatusFunc func(Status)

// Follower tails a file, handling truncation and rotation.
type Follower struct {
	Name      string
	Type      string
	Path      string
	FromStart bool
	OnStatus  StatusFunc
}

// Run watches the file until ctx is cancelled.
func (f *Follower) Run(ctx context.Context, out chan<- LineEvent) error {
	var (
		file   *os.File
		offset int64
		inode  uint64
		first  = true
	)
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()

	report := func(state State, msg string) {
		if f.OnStatus == nil {
			return
		}
		f.OnStatus(Status{
			Name:    f.Name,
			Type:    f.Type,
			Path:    f.Path,
			State:   state,
			Message: msg,
			Updated: time.Now(),
		})
	}

	open := func() error {
		if file != nil {
			_ = file.Close()
			file = nil
		}
		fh, err := os.Open(f.Path)
		if err != nil {
			return err
		}
		info, err := fh.Stat()
		if err != nil {
			_ = fh.Close()
			return err
		}
		ino, err := fileInode(info)
		if err != nil {
			_ = fh.Close()
			return err
		}

		switch {
		case first && f.FromStart:
			offset = 0
		case first:
			offset = info.Size()
		case ino != inode:
			// Rotation: new inode, read from start of new file.
			offset = 0
		case info.Size() < offset:
			// Truncation.
			offset = 0
		}
		first = false

		if _, err := fh.Seek(offset, io.SeekStart); err != nil {
			_ = fh.Close()
			return err
		}
		file = fh
		inode = ino
		report(StateOK, "watching")
		return nil
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if file == nil {
			if err := open(); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					report(StateWaiting, "waiting for file")
				} else {
					report(StateError, err.Error())
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(reopenInterval):
				}
				continue
			}
		}

		info, err := os.Stat(f.Path)
		if err != nil {
			report(StateWaiting, "waiting for file")
			_ = file.Close()
			file = nil
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(reopenInterval):
			}
			continue
		}
		ino, err := fileInode(info)
		if err != nil {
			report(StateError, err.Error())
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(reopenInterval):
			}
			continue
		}
		if ino != inode || info.Size() < offset {
			if err := open(); err != nil {
				file = nil
				continue
			}
		}

		reader := bufio.NewReader(file)
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			raw, err := reader.ReadBytes('\n')
			if len(raw) > 0 {
				offset += int64(len(raw))
				line := string(raw)
				if line[len(line)-1] == '\n' {
					line = line[:len(line)-1]
					if len(line) > 0 && line[len(line)-1] == '\r' {
						line = line[:len(line)-1]
					}
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case out <- LineEvent{Source: f.Name, Line: line, Time: time.Now()}:
				}
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				report(StateError, fmt.Sprintf("read: %v", err))
				_ = file.Close()
				file = nil
				break
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}
