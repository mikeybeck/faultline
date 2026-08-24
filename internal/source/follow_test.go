package source

import (
	"context"
	"os"
	"path/filepath"
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
