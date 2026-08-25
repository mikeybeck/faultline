package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFollowerAppendAndTruncate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	if err := os.WriteFile(path, []byte("old line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := make(chan LineEvent, 16)
	f := &Follower{Name: "t", Type: "laravel", Path: path}
	go func() { _ = f.Run(ctx, out) }()

	// Give follower time to open at EOF.
	time.Sleep(300 * time.Millisecond)

	fh, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fh.WriteString("new error\n"); err != nil {
		t.Fatal(err)
	}
	_ = fh.Close()

	select {
	case ev := <-out:
		if ev.Line != "new error" {
			t.Fatalf("got %q", ev.Line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for appended line")
	}

	// Truncate and write again.
	if err := os.WriteFile(path, []byte("after truncate\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-out:
		if ev.Line != "after truncate" {
			t.Fatalf("after truncate got %q", ev.Line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for truncated file content")
	}
}

func TestFollowerRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "error.log")
	if err := os.WriteFile(path, []byte("first\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := make(chan LineEvent, 16)
	f := &Follower{Name: "t", Type: "apache", Path: path}
	go func() { _ = f.Run(ctx, out) }()
	time.Sleep(300 * time.Millisecond)

	// Rotate: rename away and create new file at same path.
	if err := os.Rename(path, path+".1"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("rotated line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case ev := <-out:
		if ev.Line != "rotated line" {
			t.Fatalf("got %q", ev.Line)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for rotated file line")
	}
}

func TestFollowerFromStart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "laravel.log")
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := make(chan LineEvent, 16)
	f := &Follower{Name: "t", Type: "laravel", Path: path, FromStart: true}
	go func() { _ = f.Run(ctx, out) }()

	select {
	case ev := <-out:
		if ev.Line != "existing" {
			t.Fatalf("got %q", ev.Line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for from-start read")
	}
}

func TestFileIdentityStableAcrossGrowth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.log")
	if err := os.WriteFile(path, []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	id1, err := fileIdentity(f)
	if err != nil {
		t.Fatal(err)
	}
	fh, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fh.WriteString("more\n"); err != nil {
		t.Fatal(err)
	}
	_ = fh.Close()
	id2, err := pathIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatalf("identity changed after growth: %d vs %d", id1, id2)
	}
}

func TestFollowerFromStartReportsProgress(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	var b strings.Builder
	for i := 0; i < 200; i++ {
		b.WriteString("ERROR boom\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := make(chan LineEvent, 256)
	var mu sync.Mutex
	var statuses []Status
	f := &Follower{
		Name:      "t",
		Type:      "generic",
		Path:      path,
		FromStart: true,
		OnStatus: func(s Status) {
			mu.Lock()
			statuses = append(statuses, s)
			mu.Unlock()
		},
	}
	go func() { _ = f.Run(ctx, out) }()

	deadline := time.After(3 * time.Second)
	sawIngest := false
	sawWatch := false
	for !sawIngest || !sawWatch {
		select {
		case <-deadline:
			t.Fatalf("timeout; ingest=%v watch=%v statuses=%v", sawIngest, sawWatch, statusStates(statuses))
		case <-time.After(20 * time.Millisecond):
		}
		mu.Lock()
		for _, s := range statuses {
			if s.State == StateIngesting {
				sawIngest = true
			}
			if s.State == StateOK {
				sawWatch = true
			}
		}
		mu.Unlock()
	}
}

func TestFollowerAppendDoesNotReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	var b strings.Builder
	const n = 80
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "line-%d\n", i)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := make(chan LineEvent, 256)
	f := &Follower{Name: "t", Type: "generic", Path: path, FromStart: true}
	go func() { _ = f.Run(ctx, out) }()

	got := map[string]int{}
	deadline := time.After(3 * time.Second)
	for len(got) < n {
		select {
		case ev := <-out:
			got[ev.Line]++
		case <-deadline:
			t.Fatalf("timeout draining from-start, got %d/%d", len(got), n)
		}
	}

	fh, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fh.WriteString("fresh-line\n"); err != nil {
		t.Fatal(err)
	}
	_ = fh.Close()

	select {
	case ev := <-out:
		if ev.Line != "fresh-line" {
			t.Fatalf("replayed old content after append: %q", ev.Line)
		}
		if ev.Backfill {
			t.Fatal("appended line should not be backfill")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for appended line")
	}
	select {
	case ev := <-out:
		t.Fatalf("unexpected extra line after append: %q", ev.Line)
	case <-time.After(400 * time.Millisecond):
	}
}

func TestInitialOffset(t *testing.T) {
	id := uint64(7)
	off, backfill := initialOffset(false, Resume{Offset: 10, Identity: id}, id, 50)
	if off != 50 || backfill {
		t.Fatalf("tail: offset=%d backfill=%v", off, backfill)
	}
	off, backfill = initialOffset(true, Resume{}, id, 50)
	if off != 0 || !backfill {
		t.Fatalf("from start, no mark: offset=%d backfill=%v", off, backfill)
	}
	off, backfill = initialOffset(true, Resume{Offset: 20, Identity: id}, id, 50)
	if off != 20 || !backfill {
		t.Fatalf("resume mid-file: offset=%d backfill=%v", off, backfill)
	}
	off, backfill = initialOffset(true, Resume{Offset: 50, Identity: id}, id, 50)
	if off != 50 || backfill {
		t.Fatalf("resume at EOF: offset=%d backfill=%v", off, backfill)
	}
	off, backfill = initialOffset(true, Resume{Offset: 20, Identity: 99}, id, 50)
	if off != 0 || !backfill {
		t.Fatalf("identity mismatch: offset=%d backfill=%v", off, backfill)
	}
	off, backfill = initialOffset(true, Resume{Offset: 80, Identity: id}, id, 50)
	if off != 0 || !backfill {
		t.Fatalf("truncated past mark: offset=%d backfill=%v", off, backfill)
	}
}

func TestFollowerResumeSkipsPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	first := "ERROR skip-me\n"
	rest := "ERROR keep-me\n"
	if err := os.WriteFile(path, []byte(first+rest), 0o644); err != nil {
		t.Fatal(err)
	}
	fh, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	id, err := fileIdentity(fh)
	_ = fh.Close()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := make(chan LineEvent, 8)
	f := &Follower{
		Name:      "t",
		Type:      "generic",
		Path:      path,
		FromStart: true,
		Resume:    Resume{Offset: int64(len(first)), Identity: id},
	}
	go func() { _ = f.Run(ctx, out) }()

	select {
	case ev := <-out:
		if ev.Line != "ERROR keep-me" {
			t.Fatalf("got %q, want keep-me", ev.Line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for resumed line")
	}
	select {
	case ev := <-out:
		t.Fatalf("unexpected extra line: %q", ev.Line)
	case <-time.After(400 * time.Millisecond):
	}
}

func statusStates(sts []Status) []State {
	out := make([]State, len(sts))
	for i, s := range sts {
		out[i] = s.State
	}
	return out
}
