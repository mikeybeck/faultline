package source

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"
)

const (
	pollInterval     = 200 * time.Millisecond
	reopenInterval   = time.Second
	progressInterval = 250 * time.Millisecond
)

// LineEvent is a raw line read from a followed file.
type LineEvent struct {
	Source   string
	Line     string
	Time     time.Time
	Backfill bool // true while reading existing content from start
}

// StatusFunc receives source health updates.
type StatusFunc func(Status)

// Resume is a byte offset in a specific file identity (inode / file index).
type Resume struct {
	Offset   int64  `yaml:"offset" json:"offset"`
	Identity uint64 `yaml:"identity" json:"identity"`
}

// Follower tails a file, handling truncation and rotation.
type Follower struct {
	Name      string
	Type      string
	Path      string
	FromStart bool
	Resume    Resume
	OnStatus  StatusFunc

	posOff atomic.Int64
	posID  atomic.Uint64
}

// Position is the last consumed byte offset and file identity.
func (f *Follower) Position() Resume {
	return Resume{Offset: f.posOff.Load(), Identity: f.posID.Load()}
}

func (f *Follower) setPos(off int64, id uint64) {
	f.posOff.Store(off)
	if id != 0 {
		f.posID.Store(id)
	}
}

// initialOffset picks the first-open read position.
func initialOffset(fromStart bool, resume Resume, id uint64, size int64) (offset int64, backfill bool) {
	if !fromStart {
		return size, false
	}
	if resume.Identity != 0 && resume.Identity == id && resume.Offset >= 0 && resume.Offset <= size {
		return resume.Offset, resume.Offset < size
	}
	return 0, true
}

// Run watches the file until ctx is cancelled.
func (f *Follower) Run(ctx context.Context, out chan<- LineEvent) error {
	var (
		file     *os.File
		offset   int64
		inode    uint64
		first    = true
		backfill = f.FromStart
		total    int64
		lastProg time.Time
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

	reportProgress := func() {
		if !backfill {
			return
		}
		now := time.Now()
		if !lastProg.IsZero() && now.Sub(lastProg) < progressInterval && offset < total {
			return
		}
		lastProg = now
		if total < offset {
			total = offset
		}
		report(StateIngesting, fmt.Sprintf("reading %s / %s", formatBytes(offset), formatBytes(total)))
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
		id, err := fileIdentity(fh)
		if err != nil {
			_ = fh.Close()
			return err
		}

		switch {
		case first:
			offset, backfill = initialOffset(f.FromStart, f.Resume, id, info.Size())
			total = info.Size()
		case id != inode:
			// Rotation: new identity, read from start of new file.
			offset = 0
			backfill = false
			total = info.Size()
		case info.Size() < offset:
			// Truncation.
			offset = 0
			backfill = false
			total = info.Size()
		default:
			total = info.Size()
		}
		first = false

		if _, err := fh.Seek(offset, io.SeekStart); err != nil {
			_ = fh.Close()
			return err
		}
		file = fh
		inode = id
		f.setPos(offset, id)
		if backfill {
			reportProgress()
		} else {
			report(StateOK, "watching")
		}
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
		id, err := pathIdentity(f.Path)
		if err != nil {
			report(StateError, err.Error())
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(reopenInterval):
			}
			continue
		}
		if id != inode || info.Size() < offset {
			if err := open(); err != nil {
				file = nil
				continue
			}
		} else if backfill {
			total = info.Size()
		}

		reader := bufio.NewReader(file)
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			raw, err := reader.ReadBytes('\n')
			if len(raw) > 0 {
				offset += int64(len(raw))
				f.setPos(offset, inode)
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
				case out <- LineEvent{Source: f.Name, Line: line, Time: time.Now(), Backfill: backfill}:
				}
				if backfill {
					reportProgress()
				}
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					if backfill {
						backfill = false
						report(StateOK, "watching")
					}
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

func formatBytes(n int64) string {
	const mb = 1024 * 1024
	const kb = 1024
	switch {
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
