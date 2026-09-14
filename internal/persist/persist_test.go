package persist

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mikey/faultline/internal/event"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "inbox.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func sample(project string, msg string, t0 time.Time) event.Event {
	ev := event.Event{
		Source:    "browser",
		Type:      "TypeError",
		Message:   msg,
		File:      "app.js",
		Line:      9,
		Severity:  event.SeverityError,
		Time:      t0,
		FirstSeen: t0,
		LastSeen:  t0,
		Stack:     "TypeError: " + msg,
	}
	ev.Hash = event.Fingerprint(ev.Source, ev.Type, ev.Message, ev.File, ev.Line)
	return ev
}

func TestRecordLoadAndProjectIsolation(t *testing.T) {
	db := testDB(t)
	t0 := time.Now().Add(-time.Minute).Truncate(time.Millisecond)
	a := sample("/proj/a", "boom a", t0)
	b := sample("/proj/b", "boom b", t0)

	if _, err := db.Record("/proj/a", a, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Record("/proj/b", b, false); err != nil {
		t.Fatal(err)
	}

	got, err := db.LoadActive("/proj/a")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Message != "boom a" {
		t.Fatalf("project a = %+v", got)
	}
	got, err = db.LoadActive("/proj/b")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Message != "boom b" {
		t.Fatalf("project b = %+v", got)
	}
}

func TestDismissUntilNextOccurrence(t *testing.T) {
	db := testDB(t)
	t0 := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
	ev := sample("/p", "x", t0)
	if _, err := db.Record("/p", ev, false); err != nil {
		t.Fatal(err)
	}
	if err := db.Dismiss("/p", []string{ev.Hash}, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	active, err := db.LoadActive("/p")
	if err != nil || len(active) != 0 {
		t.Fatalf("dismissed still active: %+v err=%v", active, err)
	}

	same := ev
	same.Time = t0.Add(30 * time.Second)
	res, err := db.Record("/p", same, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Show {
		t.Fatal("occurrence before dismiss should stay hidden")
	}

	later := ev
	later.Time = t0.Add(2 * time.Minute)
	res, err = db.Record("/p", later, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Show || !res.IsNew {
		t.Fatalf("later occurrence should reappear: %+v", res)
	}
	active, err = db.LoadActive("/p")
	if err != nil || len(active) != 1 {
		t.Fatalf("active after reappear = %+v err=%v", active, err)
	}
}

func TestReplayDoesNotUndismiss(t *testing.T) {
	db := testDB(t)
	t0 := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
	ev := sample("/p", "x", t0)
	if _, err := db.Record("/p", ev, false); err != nil {
		t.Fatal(err)
	}
	if err := db.Dismiss("/p", []string{ev.Hash}, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	later := ev
	later.Time = t0.Add(2 * time.Hour)
	res, err := db.Record("/p", later, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Show {
		t.Fatal("log replay should not un-dismiss")
	}
}

func TestMuteTypeSuppressesNewFingerprints(t *testing.T) {
	db := testDB(t)
	t0 := time.Now().Truncate(time.Millisecond)
	if err := db.Mute("/p", KindType, "TypeError"); err != nil {
		t.Fatal(err)
	}
	ev := sample("/p", "muted", t0)
	res, err := db.Record("/p", ev, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Show {
		t.Fatal("muted type should not show")
	}
	active, err := db.LoadActive("/p")
	if err != nil || len(active) != 0 {
		t.Fatalf("load active = %+v err=%v", active, err)
	}

	other := ev
	other.Type = "ReferenceError"
	other.Message = "other"
	other.Hash = event.Fingerprint(other.Source, other.Type, other.Message, other.File, other.Line)
	res, err = db.Record("/p", other, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Show {
		t.Fatal("other type should show")
	}

	if err := db.Unmute("/p", KindType, "TypeError"); err != nil {
		t.Fatal(err)
	}
	active, err = db.LoadActive("/p")
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 2 {
		t.Fatalf("after unmute want 2, got %+v", active)
	}
}

func TestReplayDoesNotDoubleCount(t *testing.T) {
	db := testDB(t)
	t0 := time.Now().Truncate(time.Millisecond)
	ev := sample("/p", "once", t0)
	if _, err := db.Record("/p", ev, false); err != nil {
		t.Fatal(err)
	}
	replay := ev
	replay.Time = t0
	res, err := db.Record("/p", replay, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Event.Count != 1 {
		t.Fatalf("count = %d", res.Event.Count)
	}
}

func TestDeleteBySourcesKeepsOtherSources(t *testing.T) {
	db := testDB(t)
	t0 := time.Now().Truncate(time.Millisecond)
	fileEv := sample("/p", "from log", t0)
	fileEv.Source = "app"
	fileEv.Hash = event.Fingerprint(fileEv.Source, fileEv.Type, fileEv.Message, fileEv.File, fileEv.Line)
	browser := sample("/p", "from browser", t0)
	if _, err := db.Record("/p", fileEv, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Record("/p", browser, false); err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteBySources("/p", []string{"app"}); err != nil {
		t.Fatal(err)
	}
	active, err := db.LoadActive("/p")
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].Source != "browser" {
		t.Fatalf("got %+v", active)
	}
}

func TestEmptyPathIsNoop(t *testing.T) {
	db, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	ev := sample("/p", "x", time.Now())
	res, err := db.Record("/p", ev, false)
	if err != nil || !res.Show {
		t.Fatalf("noop record: %+v %v", res, err)
	}
	got, err := db.LoadActive("/p")
	if err != nil || got != nil {
		t.Fatalf("noop load: %+v %v", got, err)
	}
}
