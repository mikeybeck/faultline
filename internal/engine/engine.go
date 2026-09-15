package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mikey/faultline/internal/applog"
	"github.com/mikey/faultline/internal/config"
	"github.com/mikey/faultline/internal/editor"
	"github.com/mikey/faultline/internal/event"
	"github.com/mikey/faultline/internal/ingest"
	"github.com/mikey/faultline/internal/mark"
	"github.com/mikey/faultline/internal/notify"
	"github.com/mikey/faultline/internal/parser"
	"github.com/mikey/faultline/internal/persist"
	"github.com/mikey/faultline/internal/source"
	"github.com/mikey/faultline/internal/sourcemap"
	"github.com/mikey/faultline/internal/store"
)

const eventPingInterval = 250 * time.Millisecond

// Engine tails log sources, parses them, and aggregates events.
// While running it also listens for browser errors on 127.0.0.1:9477.
type Engine struct {
	mu sync.Mutex

	store    *store.Store
	notifier *notify.Notifier
	opener   editor.Opener
	cfg      *config.Config

	cancel    context.CancelFunc
	wg        sync.WaitGroup
	eventCh   chan event.Event
	statusCh  chan source.Status
	statuses  []source.Status
	running   bool
	marks     *mark.Store
	persist   *persist.DB
	followers []*source.Follower

	notifyCalls atomic.Int64

	projectDir   string
	ingestAddr   string
	boundIngest  string
	maps         *sourcemap.Resolver
	onNotify     func(event.Event)
	clearOnGit   bool
	extensionDir string
}

func New() *Engine {
	return &Engine{
		store:    store.New(),
		notifier: &notify.Notifier{},
		eventCh:  make(chan event.Event, 64),
		statusCh: make(chan source.Status, 64),
	}
}

func (e *Engine) Store() *store.Store { return e.store }

func (e *Engine) Events() <-chan event.Event { return e.eventCh }

func (e *Engine) Statuses() <-chan source.Status { return e.statusCh }

// NotifyCalls is the number of desktop notifications that were actually sent.
func (e *Engine) NotifyCalls() int64 { return e.notifyCalls.Load() }

func (e *Engine) Running() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

func (e *Engine) Config() *config.Config {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cfg
}

func (e *Engine) SnapshotStatuses() []source.Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]source.Status, len(e.statuses))
	copy(out, e.statuses)
	return out
}

// UseMarks attaches a persistent resume-offset store.
func (e *Engine) UseMarks(s *mark.Store) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.marks = s
}

// UsePersist attaches the SQLite inbox. Nil is a no-op (RAM only).
func (e *Engine) UsePersist(db *persist.DB) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.persist = db
}

// ClosePersist closes the inbox database.
func (e *Engine) ClosePersist() {
	e.mu.Lock()
	db := e.persist
	e.persist = nil
	e.mu.Unlock()
	if db != nil {
		_ = db.Close()
	}
}

func (e *Engine) persistState() (*persist.DB, string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.persist, e.projectDir
}

func (e *Engine) sourcePaths() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cfg == nil {
		return nil
	}
	out := make([]string, 0, len(e.cfg.Sources))
	for _, s := range e.cfg.Sources {
		if !config.IsFileSource(s.Type) {
			continue
		}
		out = append(out, s.Path)
	}
	return out
}

func (e *Engine) fileSourceNames() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cfg == nil {
		return nil
	}
	out := make([]string, 0, len(e.cfg.Sources))
	for _, s := range e.cfg.Sources {
		if !config.IsFileSource(s.Type) {
			continue
		}
		out = append(out, s.Name)
	}
	return out
}

func (e *Engine) hydrate() {
	db, project := e.persistState()
	if db == nil || project == "" {
		return
	}
	items, err := db.LoadActive(project)
	if err != nil {
		return
	}
	e.store.Load(items)
}

// Positions returns the current read offset of each followed file.
func (e *Engine) Positions() map[string]source.Resume {
	e.mu.Lock()
	fs := e.followers
	e.mu.Unlock()
	out := make(map[string]source.Resume, len(fs))
	for _, f := range fs {
		if f == nil {
			continue
		}
		r := f.Position()
		if r.Identity == 0 {
			continue
		}
		out[f.Path] = r
	}
	return out
}

// Clear dismisses the inbox (until the next occurrence) and saves log resume marks.
func (e *Engine) Clear() {
	if e.marks != nil {
		_ = e.marks.PutMany(e.Positions())
	}
	if db, project := e.persistState(); db != nil {
		_ = db.DismissAll(project, time.Now())
	}
	e.store.Clear()
}

// ResetInbox drops in-memory events without changing persistence or marks.
func (e *Engine) ResetInbox() {
	e.store.Clear()
}

// ClearMarks forgets saved resume points and drops file-source events so logs can be re-read.
// Browser events, mutes, and dismissed browser rows are left alone.
func (e *Engine) ClearMarks() {
	names := e.fileSourceNames()
	e.store.RemoveBySources(names...)
	if db, project := e.persistState(); db != nil {
		_ = db.DeleteBySources(project, names)
	}
	if e.marks != nil {
		_ = e.marks.Delete(e.sourcePaths()...)
	}
}

// Dismiss hides fingerprints until they occur again.
func (e *Engine) Dismiss(hashes []string) {
	if len(hashes) == 0 {
		return
	}
	if db, project := e.persistState(); db != nil {
		_ = db.Dismiss(project, hashes, time.Now())
	}
	e.store.Remove(hashes...)
}

// DismissMatching hides events that match the current inbox filters.
func (e *Engine) DismissMatching(filter, severity, source, typ string) {
	e.Dismiss(e.store.MatchingHashes(filter, severity, source, typ))
}

// Mute hides a fingerprint or type until unmuted.
func (e *Engine) Mute(kind, value string) error {
	if db, project := e.persistState(); db != nil {
		if err := db.Mute(project, kind, value); err != nil {
			return err
		}
	}
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case persist.KindHash:
		e.store.Remove(strings.TrimSpace(value))
	case persist.KindType:
		e.store.Remove(e.store.MatchingHashes("", "all", "", value)...)
	}
	return nil
}

// Unmute removes a hide rule and restores matching active events.
func (e *Engine) Unmute(kind, value string) error {
	db, project := e.persistState()
	if db == nil {
		return nil
	}
	if err := db.Unmute(project, kind, value); err != nil {
		return err
	}
	e.hydrate()
	return nil
}

// Mutes returns hide rules for the current project.
func (e *Engine) Mutes() []persist.Mute {
	db, project := e.persistState()
	if db == nil {
		return nil
	}
	m, err := db.Mutes(project)
	if err != nil {
		return nil
	}
	return m
}

func (e *Engine) SetExtensionDir(dir string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.extensionDir = dir
}

func (e *Engine) SetNotifyActivate(fn func(event.Event)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onNotify = fn
	if e.notifier != nil {
		e.notifier.OnActivate = fn
	}
}

func (e *Engine) Snooze(hashes []string, d time.Duration) {
	if len(hashes) == 0 || d <= 0 {
		return
	}
	until := time.Now().Add(d)
	if db, project := e.persistState(); db != nil {
		_ = db.Snooze(project, hashes, until)
	}
	e.store.Remove(hashes...)
	e.pingInbox()
}

// HasMarks reports whether any current source has a saved resume point.
func (e *Engine) HasMarks() bool {
	e.mu.Lock()
	s := e.marks
	e.mu.Unlock()
	if s == nil {
		return false
	}
	return s.HasAny(e.sourcePaths()...)
}

func (e *Engine) Open(file string, line int) error {
	e.mu.Lock()
	opener := e.opener
	e.mu.Unlock()
	return opener.Open(file, line)
}

func (e *Engine) SetNotifier(enabled, sound bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.notifier.Enabled = enabled
	e.notifier.Sound = sound
}

func (e *Engine) SetEditor(command string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.opener.Command = command
}

// SetProjectDir is used to map browser stack URLs onto project files.
func (e *Engine) SetProjectDir(dir string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.projectDir = dir
}

// SetIngestAddr overrides the browser HTTP listen address.
// ingest.Disabled skips the listener (tests).
func (e *Engine) SetIngestAddr(addr string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ingestAddr = addr
}

// BoundIngest is the actual host:port of the browser listener, if running.
func (e *Engine) BoundIngest() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.boundIngest
}

// Start begins watching cfg. Stop any previous run first.
func (e *Engine) Start(parent context.Context, cfg *config.Config, fromStart bool) error {
	if cfg == nil {
		return fmt.Errorf("engine: nil config")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	e.Stop()

	e.mu.Lock()
	ingestAddr := e.resolveIngestAddrLocked(cfg)
	projectDir := e.projectDir
	extraHosts := append([]string(nil), cfg.Browser.ExtraHosts...)
	extDir := e.extensionDir
	e.boundIngest = ""
	e.mu.Unlock()

	parsers := make(map[string]parser.Parser, len(cfg.Sources))
	initial := make([]source.Status, 0, len(cfg.Sources)+1)
	browserName := ingest.DefaultName
	hasBrowser := false
	for _, src := range cfg.Sources {
		if src.Type == "browser" {
			hasBrowser = true
			if src.Name != "" {
				browserName = src.Name
			}
			initial = append(initial, source.Status{
				Name:  src.Name,
				Type:  src.Type,
				Path:  ingest.NormalizeAddr(src.Path),
				State: source.StateWaiting,
			})
			continue
		}
		ptype := src.Type
		if src.Type == "command" {
			ptype = src.Parser
			if ptype == "" {
				ptype = "generic"
			}
		}
		p, err := parser.ForType(ptype, src.Name)
		if err != nil {
			return err
		}
		parsers[src.Name] = p
		state := source.StateWaiting
		if src.Type == "command" {
			state = source.StateWaiting
		}
		initial = append(initial, source.Status{
			Name:  src.Name,
			Type:  src.Type,
			Path:  src.Path,
			State: state,
		})
	}
	if ingestAddr != ingest.Disabled && !hasBrowser {
		initial = append(initial, source.Status{
			Name:  browserName,
			Type:  "browser",
			Path:  ingest.NormalizeAddr(ingestAddr),
			State: source.StateWaiting,
		})
	}

	ctx, cancel := context.WithCancel(parent)
	lineCh := make(chan source.LineEvent, 256)
	parsedCh := make(chan event.Event, 64)

	e.mu.Lock()
	e.cfg = cfg
	e.cancel = cancel
	e.statuses = initial
	e.opener.Command = cfg.Editor.Command
	e.notifier.Enabled = cfg.Notifications.Enabled
	e.notifier.Sound = cfg.Notifications.Sound
	e.running = true
	e.notifyCalls.Store(0)
	e.maps = sourcemap.NewResolver(projectDir)
	e.clearOnGit = cfg.Inbox.ClearOnCommit
	e.notifier.OnActivate = e.onNotify
	marks := e.marks
	e.mu.Unlock()

	e.hydrate()
	e.pingInbox()

	followers := make([]*source.Follower, 0, len(cfg.Sources))
	for _, src := range cfg.Sources {
		if src.Type == "browser" {
			continue
		}
		onStatus := func(s source.Status) {
			e.noteStatus(s)
			select {
			case e.statusCh <- s:
			default:
			}
		}
		if src.Type == "command" {
			cmd := &source.Command{
				Name:     src.Name,
				Type:     src.Type,
				Path:     src.Path,
				Dir:      projectDir,
				OnStatus: onStatus,
			}
			e.wg.Add(1)
			go func(c *source.Command) {
				defer e.wg.Done()
				_ = c.Run(ctx, lineCh)
			}(cmd)
			continue
		}
		var resume source.Resume
		if marks != nil && fromStart {
			if r, ok := marks.Get(src.Path); ok {
				resume = r
			}
		}
		f := &source.Follower{
			Name:      src.Name,
			Type:      src.Type,
			Path:      src.Path,
			FromStart: fromStart,
			Resume:    resume,
			OnStatus:  onStatus,
		}
		followers = append(followers, f)
	}
	e.mu.Lock()
	e.followers = followers
	e.mu.Unlock()

	for _, f := range followers {
		f := f
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			_ = f.Run(ctx, lineCh)
		}()
	}

	if ingestAddr != ingest.Disabled {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			_, _ = ingest.Serve(ctx, ingestAddr, ingest.Options{
				Name:         browserName,
				ProjectDir:   projectDir,
				ExtraHosts:   extraHosts,
				ExtensionDir: extDir,
				Emit: func(ev event.Event) {
					select {
					case parsedCh <- ev:
					case <-ctx.Done():
					}
				},
				OnStatus: func(s source.Status) {
					e.noteStatus(s)
					select {
					case e.statusCh <- s:
					default:
					}
				},
				Ready: func(addr string) {
					e.mu.Lock()
					e.boundIngest = addr
					e.mu.Unlock()
				},
			})
		}()
	}

	e.wg.Add(1)
	go e.ingestLoop(ctx, parsers, lineCh, parsedCh)
	if cfg.Inbox.ClearOnCommit {
		e.wg.Add(1)
		go e.watchGitHEAD(ctx, projectDir)
	}
	return nil
}

func (e *Engine) resolveIngestAddrLocked(cfg *config.Config) string {
	if e.ingestAddr != "" {
		return e.ingestAddr
	}
	for _, s := range cfg.Sources {
		if s.Type == "browser" {
			if strings.TrimSpace(s.Path) != "" {
				return ingest.NormalizeAddr(s.Path)
			}
			return ingest.DefaultAddr
		}
	}
	return ingest.DefaultAddr
}

// Restart stops watchers and starts again with cfg.
func (e *Engine) Restart(parent context.Context, cfg *config.Config, fromStart bool) error {
	return e.Start(parent, cfg, fromStart)
}

// Stop cancels watchers and waits briefly for them to exit.
func (e *Engine) Stop() {
	e.mu.Lock()
	cancel := e.cancel
	running := e.running
	e.cancel = nil
	e.running = false
	e.boundIngest = ""
	e.mu.Unlock()
	if !running && cancel == nil {
		return
	}
	if cancel != nil {
		cancel()
	}
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(750 * time.Millisecond):
	}
}

func (e *Engine) ingestLoop(ctx context.Context, parsers map[string]parser.Parser, lineCh <-chan source.LineEvent, parsedCh <-chan event.Event) {
	defer e.wg.Done()
	flushTicker := time.NewTicker(250 * time.Millisecond)
	defer flushTicker.Stop()
	pingTicker := time.NewTicker(eventPingInterval)
	defer pingTicker.Stop()
	var lastActivity time.Time
	var lastEvent event.Event
	dirty := false
	backfillSeen := false

	ping := func() {
		if !dirty {
			return
		}
		select {
		case e.eventCh <- lastEvent:
			dirty = false
		default:
		}
	}

	ingest := func(ev *event.Event, backfill bool) {
		if ev == nil {
			return
		}
		e.mu.Lock()
		maps := e.maps
		e.mu.Unlock()
		sourcemap.Apply(maps, ev, 0)
		db, project := e.persistState()
		if db != nil && project != "" {
			res, err := db.Record(project, *ev, backfill)
			if err != nil {
				return
			}
			if !res.Show {
				e.store.Remove(res.Event.Hash)
				return
			}
			e.store.Put(res.Event)
			lastEvent = res.Event
			if backfill {
				backfillSeen = true
				return
			}
			if backfillSeen {
				backfillSeen = false
				dirty = true
			}
			if res.IsNew {
				e.mu.Lock()
				n := e.notifier
				e.mu.Unlock()
				e.notifyCalls.Add(1)
				n.NewError(res.Event)
			}
			dirty = true
			return
		}
		res := e.store.Ingest(*ev)
		lastEvent = res.Event
		if backfill {
			backfillSeen = true
			return
		}
		if backfillSeen {
			backfillSeen = false
			dirty = true
		}
		if res.IsNew {
			e.mu.Lock()
			n := e.notifier
			e.mu.Unlock()
			e.notifyCalls.Add(1)
			n.NewError(res.Event)
		}
		dirty = true
	}

	flushAll := func(backfill bool) {
		for _, p := range parsers {
			for _, ev := range p.Flush() {
				ingest(ev, backfill)
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			flushAll(false)
			ping()
			return
		case <-flushTicker.C:
			if !lastActivity.IsZero() && time.Since(lastActivity) >= 750*time.Millisecond {
				flushAll(backfillSeen)
				if backfillSeen {
					backfillSeen = false
					dirty = true
					ping()
				}
				lastActivity = time.Time{}
			}
		case <-pingTicker.C:
			if !backfillSeen {
				ping()
			}
		case ev, ok := <-parsedCh:
			if !ok {
				flushAll(false)
				ping()
				return
			}
			ingest(&ev, false)
		case line, ok := <-lineCh:
			if !ok {
				flushAll(false)
				ping()
				return
			}
			lastActivity = time.Now()
			p := parsers[line.Source]
			if p == nil {
				continue
			}
			for _, ev := range p.Feed(line.Line) {
				ingest(ev, line.Backfill)
			}
			if !line.Backfill && backfillSeen {
				flushAll(false)
				backfillSeen = false
				dirty = true
				ping()
			}
		}
	}
}

func (e *Engine) noteStatus(s source.Status) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := range e.statuses {
		if e.statuses[i].Name == s.Name {
			e.statuses[i] = s
			return
		}
	}
	e.statuses = append(e.statuses, s)
}

func (e *Engine) watchGitHEAD(ctx context.Context, projectDir string) {
	defer e.wg.Done()
	path := gitHEADFile(projectDir)
	if path == "" {
		return
	}
	last, _ := os.ReadFile(path)
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			cur, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if string(cur) == string(last) {
				continue
			}
			last = cur
			e.Clear()
			e.pingInbox()
		}
	}
}

func gitHEADFile(projectDir string) string {
	if projectDir == "" {
		return ""
	}
	git := filepath.Join(projectDir, ".git")
	st, err := os.Stat(git)
	if err != nil {
		return ""
	}
	if st.IsDir() {
		return filepath.Join(git, "HEAD")
	}
	data, err := os.ReadFile(git)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(data))
	const p = "gitdir:"
	if !strings.HasPrefix(strings.ToLower(line), p) {
		return ""
	}
	dir := strings.TrimSpace(line[len(p):])
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(projectDir, dir)
	}
	return filepath.Join(dir, "HEAD")
}

func (e *Engine) pingInbox() {
	items := e.store.List("")
	var ev event.Event
	if len(items) > 0 {
		ev = items[0]
	}
	select {
	case e.eventCh <- ev:
	default:
	}
}

// ReportInternal records a Faultline-own error to the log file and the inbox.
func (e *Engine) ReportInternal(message, stack string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	_ = applog.Write(message, stack)
	now := time.Now()
	logPath, _ := applog.Path()
	ev := event.Event{
		Source:    applog.Source,
		Time:      now,
		Type:      "Faultline",
		Message:   message,
		File:      logPath,
		Severity:  event.SeverityError,
		Stack:     strings.TrimSpace(stack),
		Raw:       strings.TrimSpace(message + "\n" + stack),
		Count:     1,
		FirstSeen: now,
		LastSeen:  now,
	}
	ev.Hash = event.Fingerprint(ev.Source, ev.Type, ev.Message, filepath.Base(ev.File), 0)

	db, project := e.persistState()
	if project == "" {
		project = applog.Source
	}
	if db != nil {
		res, err := db.Record(project, ev, false)
		if err == nil {
			if res.Show {
				e.store.Put(res.Event)
			}
		} else {
			e.store.Ingest(ev)
		}
	} else {
		e.store.Ingest(ev)
	}
	st := source.Status{
		Name:    applog.Source,
		Type:    "generic",
		Path:    logPath,
		State:   source.StateError,
		Message: message,
		Updated: now,
	}
	e.noteStatus(st)
	select {
	case e.statusCh <- st:
	default:
	}
	e.pingInbox()
}
