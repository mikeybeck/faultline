package persist

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mikey/faultline/internal/event"
	_ "modernc.org/sqlite"
)

const (
	KindHash = "hash"
	KindType = "type"
)

// Mute is a durable hide rule for a fingerprint or event type.
type Mute struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// RecordResult is the inbox decision after persisting an occurrence.
type RecordResult struct {
	Event event.Event
	Show  bool
	IsNew bool
}

// DB is a per-app SQLite inbox. Rows are keyed by project path.
type DB struct {
	mu  sync.Mutex
	sql *sql.DB
}

// Default opens ~/.config/faultline/inbox.db (or the OS equivalent).
func Default() (*DB, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("inbox db: %w", err)
	}
	return Open(filepath.Join(base, "faultline", "inbox.db"))
}

// Open creates or opens the database at path. An empty path is a no-op store.
func Open(path string) (*DB, error) {
	if strings.TrimSpace(path) == "" {
		return &DB{}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("inbox db: %w", err)
	}
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("inbox db: %w", err)
	}
	sqldb.SetMaxOpenConns(1)
	d := &DB{sql: sqldb}
	if err := d.migrate(); err != nil {
		_ = sqldb.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) live() bool {
	return d != nil && d.sql != nil
}

func (d *DB) Close() error {
	if !d.live() {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	err := d.sql.Close()
	d.sql = nil
	return err
}

func (d *DB) migrate() error {
	_, err := d.sql.Exec(`
CREATE TABLE IF NOT EXISTS events (
  project TEXT NOT NULL,
  hash TEXT NOT NULL,
  source TEXT NOT NULL,
  type TEXT NOT NULL,
  message TEXT NOT NULL,
  file TEXT NOT NULL,
  line INTEGER NOT NULL,
  severity TEXT NOT NULL,
  stack TEXT,
  raw TEXT,
  count INTEGER NOT NULL,
  first_seen INTEGER NOT NULL,
  last_seen INTEGER NOT NULL,
  dismissed_at INTEGER,
  snippet TEXT,
  context TEXT,
  samples TEXT,
  PRIMARY KEY (project, hash)
);
CREATE INDEX IF NOT EXISTS idx_events_project_seen ON events(project, last_seen);
CREATE TABLE IF NOT EXISTS mutes (
  project TEXT NOT NULL,
  kind TEXT NOT NULL,
  value TEXT NOT NULL,
  PRIMARY KEY (project, kind, value)
);
`)
	if err != nil {
		return fmt.Errorf("inbox db migrate: %w", err)
	}
	_, _ = d.sql.Exec(`ALTER TABLE events ADD COLUMN snoozed_until INTEGER`)
	_, _ = d.sql.Exec(`ALTER TABLE events ADD COLUMN snippet TEXT`)
	_, _ = d.sql.Exec(`ALTER TABLE events ADD COLUMN context TEXT`)
	_, _ = d.sql.Exec(`ALTER TABLE events ADD COLUMN samples TEXT`)
	return nil
}

type row struct {
	ev           event.Event
	dismissedAt  sql.NullInt64
	snoozedUntil sql.NullInt64
}

// Record upserts an occurrence. replay skips count bumps for hashes already stored
// (log backfill after hydrate) but still inserts fingerprints that appeared while down.
func (d *DB) Record(project string, ev event.Event, replay bool) (RecordResult, error) {
	if !d.live() || project == "" {
		return RecordResult{Event: ev, Show: true, IsNew: true}, nil
	}
	project = key(project)
	normalize(&ev)

	d.mu.Lock()
	defer d.mu.Unlock()

	existing, err := d.getLocked(project, ev.Hash)
	if err != nil {
		return RecordResult{}, err
	}

	now := ev.Time
	if now.IsZero() {
		now = time.Now()
		ev.Time = now
	}

	muted, err := d.isMutedLocked(project, ev.Hash, ev.Type)
	if err != nil {
		return RecordResult{}, err
	}

	if existing == nil {
		if ev.Count <= 0 {
			ev.Count = 1
		}
		if ev.FirstSeen.IsZero() {
			ev.FirstSeen = now
		}
		if ev.LastSeen.IsZero() {
			ev.LastSeen = now
		}
		ev.Samples = event.PushSample(ev.Samples, ev.Message)
		if err := d.insertLocked(project, ev, nil); err != nil {
			return RecordResult{}, err
		}
		return RecordResult{Event: ev, Show: !muted, IsNew: !muted}, nil
	}

	merged := existing.ev
	if replay {
		mergePayload(&merged, ev, now, false)
	} else {
		merged.Count++
		mergePayload(&merged, ev, now, true)
	}

	reappeared := false
	var dismissed *time.Time
	if existing.dismissedAt.Valid {
		at := msTime(existing.dismissedAt.Int64)
		// Replaying a log is not a new occurrence — only live ingest can un-dismiss.
		if !replay && now.After(at) {
			reappeared = true
		} else {
			dismissed = &at
		}
	}
	if err := d.updateLocked(project, merged, dismissed); err != nil {
		return RecordResult{}, err
	}
	snoozed := existing.snoozedUntil.Valid && now.Before(msTime(existing.snoozedUntil.Int64))
	show := !muted && dismissed == nil && !snoozed
	return RecordResult{Event: merged, Show: show, IsNew: show && reappeared}, nil
}

// LoadActive returns unmuted, non-dismissed events for project.
func (d *DB) LoadActive(project string) ([]event.Event, error) {
	if !d.live() || project == "" {
		return nil, nil
	}
	project = key(project)
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.sql.Query(`
SELECT e.hash, e.source, e.type, e.message, e.file, e.line, e.severity, e.stack, e.raw,
       e.count, e.first_seen, e.last_seen, e.snippet, e.context, e.samples
FROM events e
WHERE e.project = ?
  AND e.dismissed_at IS NULL
  AND (e.snoozed_until IS NULL OR e.snoozed_until <= ?)
  AND NOT EXISTS (
    SELECT 1 FROM mutes m
    WHERE m.project = e.project
      AND (
        (m.kind = ? AND m.value = e.hash)
        OR (m.kind = ? AND lower(m.value) = lower(e.type))
      )
  )
ORDER BY e.last_seen DESC
`, project, timeMS(time.Now()), KindHash, KindType)
	if err != nil {
		return nil, fmt.Errorf("inbox load: %w", err)
	}
	defer rows.Close()

	var out []event.Event
	for rows.Next() {
		ev, err := scanActive(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

// Dismiss hides hashes until a later occurrence.
func (d *DB) Dismiss(project string, hashes []string, at time.Time) error {
	if !d.live() || project == "" || len(hashes) == 0 {
		return nil
	}
	project = key(project)
	if at.IsZero() {
		at = time.Now()
	}
	ms := timeMS(at)
	d.mu.Lock()
	defer d.mu.Unlock()
	tx, err := d.sql.Begin()
	if err != nil {
		return fmt.Errorf("inbox dismiss: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(`UPDATE events SET dismissed_at = ? WHERE project = ? AND hash = ?`)
	if err != nil {
		return fmt.Errorf("inbox dismiss: %w", err)
	}
	defer stmt.Close()
	for _, h := range hashes {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if _, err := stmt.Exec(ms, project, h); err != nil {
			return fmt.Errorf("inbox dismiss: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("inbox dismiss: %w", err)
	}
	return nil
}

// Snooze hides hashes until until, including new occurrences in that window.
func (d *DB) Snooze(project string, hashes []string, until time.Time) error {
	if !d.live() || project == "" || len(hashes) == 0 {
		return nil
	}
	project = key(project)
	if until.IsZero() {
		until = time.Now().Add(15 * time.Minute)
	}
	ms := timeMS(until)
	d.mu.Lock()
	defer d.mu.Unlock()
	tx, err := d.sql.Begin()
	if err != nil {
		return fmt.Errorf("inbox snooze: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(`UPDATE events SET snoozed_until = ? WHERE project = ? AND hash = ?`)
	if err != nil {
		return fmt.Errorf("inbox snooze: %w", err)
	}
	defer stmt.Close()
	for _, h := range hashes {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if _, err := stmt.Exec(ms, project, h); err != nil {
			return fmt.Errorf("inbox snooze: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("inbox snooze: %w", err)
	}
	return nil
}

// DismissAll marks every active event in the project as dismissed.
func (d *DB) DismissAll(project string, at time.Time) error {
	if !d.live() || project == "" {
		return nil
	}
	project = key(project)
	if at.IsZero() {
		at = time.Now()
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.sql.Exec(
		`UPDATE events SET dismissed_at = ? WHERE project = ? AND dismissed_at IS NULL`,
		timeMS(at), project,
	)
	if err != nil {
		return fmt.Errorf("inbox dismiss all: %w", err)
	}
	return nil
}

// DeleteBySources removes events whose source name is in names (log replay).
func (d *DB) DeleteBySources(project string, names []string) error {
	if !d.live() || project == "" || len(names) == 0 {
		return nil
	}
	project = key(project)
	d.mu.Lock()
	defer d.mu.Unlock()
	tx, err := d.sql.Begin()
	if err != nil {
		return fmt.Errorf("inbox delete sources: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(`DELETE FROM events WHERE project = ? AND source = ?`)
	if err != nil {
		return fmt.Errorf("inbox delete sources: %w", err)
	}
	defer stmt.Close()
	for _, name := range names {
		if _, err := stmt.Exec(project, name); err != nil {
			return fmt.Errorf("inbox delete sources: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("inbox delete sources: %w", err)
	}
	return nil
}

// Mute adds a hide rule.
func (d *DB) Mute(project, kind, value string) error {
	if !d.live() || project == "" {
		return nil
	}
	kind, value, ok := normalizeMute(kind, value)
	if !ok {
		return fmt.Errorf("inbox mute: unknown kind %q", kind)
	}
	project = key(project)
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.sql.Exec(
		`INSERT OR REPLACE INTO mutes (project, kind, value) VALUES (?, ?, ?)`,
		project, kind, value,
	)
	if err != nil {
		return fmt.Errorf("inbox mute: %w", err)
	}
	return nil
}

// Unmute removes a hide rule.
func (d *DB) Unmute(project, kind, value string) error {
	if !d.live() || project == "" {
		return nil
	}
	kind, value, ok := normalizeMute(kind, value)
	if !ok {
		return nil
	}
	project = key(project)
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.sql.Exec(`DELETE FROM mutes WHERE project = ? AND kind = ? AND value = ?`, project, kind, value)
	if err != nil {
		return fmt.Errorf("inbox unmute: %w", err)
	}
	return nil
}

// Mutes lists hide rules for the project.
func (d *DB) Mutes(project string) ([]Mute, error) {
	if !d.live() || project == "" {
		return nil, nil
	}
	project = key(project)
	d.mu.Lock()
	defer d.mu.Unlock()
	rows, err := d.sql.Query(`SELECT kind, value FROM mutes WHERE project = ? ORDER BY kind, value`, project)
	if err != nil {
		return nil, fmt.Errorf("inbox mutes: %w", err)
	}
	defer rows.Close()
	var out []Mute
	for rows.Next() {
		var m Mute
		if err := rows.Scan(&m.Kind, &m.Value); err != nil {
			return nil, fmt.Errorf("inbox mutes: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (d *DB) getLocked(project, hash string) (*row, error) {
	r := d.sql.QueryRow(`
SELECT hash, source, type, message, file, line, severity, stack, raw, count, first_seen, last_seen,
       snippet, context, samples, dismissed_at
FROM events WHERE project = ? AND hash = ?`, project, hash)
	ev, dismissed, err := scanStored(r)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inbox get: %w", err)
	}
	return &row{ev: ev, dismissedAt: dismissed, snoozedUntil: d.snoozedLocked(project, hash)}, nil
}

func (d *DB) snoozedLocked(project, hash string) sql.NullInt64 {
	var until sql.NullInt64
	_ = d.sql.QueryRow(`SELECT snoozed_until FROM events WHERE project = ? AND hash = ?`, project, hash).Scan(&until)
	return until
}

func (d *DB) isMutedLocked(project, hash, typ string) (bool, error) {
	var n int
	err := d.sql.QueryRow(`
SELECT COUNT(*) FROM mutes WHERE project = ?
  AND ((kind = ? AND value = ?) OR (kind = ? AND lower(value) = lower(?)))
`, project, KindHash, hash, KindType, typ).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("inbox muted: %w", err)
	}
	return n > 0, nil
}

func (d *DB) insertLocked(project string, ev event.Event, dismissed *time.Time) error {
	var dismissedMS any
	if dismissed != nil {
		dismissedMS = timeMS(*dismissed)
	}
	_, err := d.sql.Exec(`
INSERT INTO events (project, hash, source, type, message, file, line, severity, stack, raw, count, first_seen, last_seen, dismissed_at, snippet, context, samples)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		project, ev.Hash, ev.Source, ev.Type, ev.Message, ev.File, ev.Line, string(ev.Severity),
		ev.Stack, ev.Raw, ev.Count, timeMS(ev.FirstSeen), timeMS(ev.LastSeen), dismissedMS,
		nullString(ev.Snippet), marshalStrings(ev.Context), marshalStrings(ev.Samples),
	)
	if err != nil {
		return fmt.Errorf("inbox insert: %w", err)
	}
	return nil
}

func (d *DB) updateLocked(project string, ev event.Event, dismissed *time.Time) error {
	var dismissedMS any
	if dismissed != nil {
		dismissedMS = timeMS(*dismissed)
	}
	_, err := d.sql.Exec(`
UPDATE events SET source=?, type=?, message=?, file=?, line=?, severity=?, stack=?, raw=?,
  count=?, first_seen=?, last_seen=?, dismissed_at=?, snippet=?, context=?, samples=?
WHERE project=? AND hash=?`,
		ev.Source, ev.Type, ev.Message, ev.File, ev.Line, string(ev.Severity), ev.Stack, ev.Raw,
		ev.Count, timeMS(ev.FirstSeen), timeMS(ev.LastSeen), dismissedMS,
		nullString(ev.Snippet), marshalStrings(ev.Context), marshalStrings(ev.Samples),
		project, ev.Hash,
	)
	if err != nil {
		return fmt.Errorf("inbox update: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanActive(s scanner) (event.Event, error) {
	ev, _, err := scanCols(s, false)
	return ev, err
}

func scanStored(s scanner) (event.Event, sql.NullInt64, error) {
	return scanCols(s, true)
}

func scanCols(s scanner, withDismissed bool) (event.Event, sql.NullInt64, error) {
	var ev event.Event
	var sev string
	var first, last int64
	var dismissed sql.NullInt64
	var stack, raw, snippet, contextJSON, samplesJSON sql.NullString
	dest := []any{
		&ev.Hash, &ev.Source, &ev.Type, &ev.Message, &ev.File, &ev.Line, &sev,
		&stack, &raw, &ev.Count, &first, &last, &snippet, &contextJSON, &samplesJSON,
	}
	if withDismissed {
		dest = append(dest, &dismissed)
	}
	if err := s.Scan(dest...); err != nil {
		return event.Event{}, sql.NullInt64{}, err
	}
	ev.Severity = event.Severity(sev)
	if stack.Valid {
		ev.Stack = stack.String
	}
	if raw.Valid {
		ev.Raw = raw.String
	}
	if snippet.Valid {
		ev.Snippet = snippet.String
	}
	ev.Context = unmarshalStrings(contextJSON)
	ev.Samples = unmarshalStrings(samplesJSON)
	ev.FirstSeen = msTime(first)
	ev.LastSeen = msTime(last)
	ev.Time = ev.LastSeen
	return ev, dismissed, nil
}

func mergePayload(dst *event.Event, ev event.Event, now time.Time, live bool) {
	if live {
		dst.LastSeen = now
		if now.After(dst.Time) {
			dst.Time = now
		}
		dst.Message = ev.Message
		dst.Samples = event.PushSample(dst.Samples, ev.Message)
	} else if now.After(dst.LastSeen) {
		dst.LastSeen = now
		dst.Time = now
	}
	if ev.Stack != "" {
		dst.Stack = ev.Stack
	}
	if ev.Raw != "" {
		dst.Raw = ev.Raw
	}
	if ev.Snippet != "" {
		dst.Snippet = ev.Snippet
	}
	if len(ev.Context) > 0 {
		dst.Context = append([]string{}, ev.Context...)
	}
	if dst.File == "" && ev.File != "" {
		dst.File = ev.File
		dst.Line = ev.Line
	}
}

func marshalStrings(v []string) any {
	if len(v) == 0 {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return string(b)
}

func unmarshalStrings(ns sql.NullString) []string {
	if !ns.Valid || strings.TrimSpace(ns.String) == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(ns.String), &out); err != nil {
		return nil
	}
	return out
}

func nullString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func normalize(ev *event.Event) {
	if ev.Hash == "" {
		ev.Hash = event.Fingerprint(ev.Source, ev.Type, ev.Message, ev.File, ev.Line)
	}
	if ev.Count == 0 {
		ev.Count = 1
	}
	if ev.Time.IsZero() {
		ev.Time = time.Now()
	}
	if ev.FirstSeen.IsZero() {
		ev.FirstSeen = ev.Time
	}
	if ev.LastSeen.IsZero() {
		ev.LastSeen = ev.Time
	}
}

func normalizeMute(kind, value string) (string, string, bool) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", false
	}
	switch kind {
	case KindHash, KindType:
		return kind, value, true
	default:
		return kind, value, false
	}
}

func key(project string) string {
	project = filepath.Clean(project)
	abs, err := filepath.Abs(project)
	if err != nil {
		return project
	}
	return abs
}

func timeMS(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func msTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}
