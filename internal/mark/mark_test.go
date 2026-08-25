package mark

import (
	"path/filepath"
	"testing"

	"github.com/mikey/faultline/internal/source"
)

func TestPutGetDelete(t *testing.T) {
	dir := t.TempDir()
	s := Open(filepath.Join(dir, "marks.yaml"))
	path := filepath.Join(dir, "app.log")

	if _, ok := s.Get(path); ok {
		t.Fatal("expected no mark")
	}
	if s.HasAny(path) {
		t.Fatal("HasAny should be false")
	}

	want := source.Resume{Offset: 42, Identity: 99}
	if err := s.PutMany(map[string]source.Resume{path: want}); err != nil {
		t.Fatal(err)
	}
	got, ok := s.Get(path)
	if !ok {
		t.Fatal("expected mark")
	}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
	if !s.HasAny(path) {
		t.Fatal("HasAny should be true")
	}

	if err := s.Delete(path); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get(path); ok {
		t.Fatal("mark should be gone")
	}
}

func TestSkipZeroIdentity(t *testing.T) {
	dir := t.TempDir()
	s := Open(filepath.Join(dir, "marks.yaml"))
	path := filepath.Join(dir, "app.log")
	if err := s.PutMany(map[string]source.Resume{path: {Offset: 10, Identity: 0}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get(path); ok {
		t.Fatal("zero identity should not be stored")
	}
}

func TestNoopStore(t *testing.T) {
	s := Open("")
	if err := s.PutMany(map[string]source.Resume{"x": {Offset: 1, Identity: 1}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get("x"); ok {
		t.Fatal("noop store should not persist")
	}
}
