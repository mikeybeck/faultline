package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mikey/faultline/internal/config"
	"github.com/mikey/faultline/internal/editor"
	"github.com/mikey/faultline/internal/event"
	"github.com/mikey/faultline/internal/mark"
	"github.com/mikey/faultline/internal/notify"
	"github.com/mikey/faultline/internal/parser"
	"github.com/mikey/faultline/internal/source"
	"github.com/mikey/faultline/internal/store"
)

const eventPingInterval = 250 * time.Millisecond

// Engine tails log sources, parses them, and aggregates events.
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
	followers []*source.Follower

	notifyCalls atomic.Int64
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

func (e *Engine) sourcePaths() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cfg == nil {
		return nil
	}
	out := make([]string, 0, len(e.cfg.Sources))
	for _, s := range e.cfg.Sources {
		out = append(out, s.Path)
	}
	return out
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

// Clear empties the inbox and saves resume marks at the current file offsets.
func (e *Engine) Clear() {
	if e.marks != nil {
		_ = e.marks.PutMany(e.Positions())
	}
	e.store.Clear()
}

// ClearMarks forgets saved resume points and empties the inbox.
func (e *Engine) ClearMarks() {
	e.store.Clear()
	if e.marks != nil {
		_ = e.marks.Delete(e.sourcePaths()...)
	}
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

// Start begins watching cfg. Stop any previous run first.
func (e *Engine) Start(parent context.Context, cfg *config.Config, fromStart bool) error {
	if cfg == nil {
		return fmt.Errorf("engine: nil config")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	e.Stop()

	parsers := make(map[string]parser.Parser, len(cfg.Sources))
	initial := make([]source.Status, 0, len(cfg.Sources))
	for _, src := range cfg.Sources {
		p, err := parser.ForType(src.Type, src.Name)
		if err != nil {
			return err
		}
		parsers[src.Name] = p
		initial = append(initial, source.Status{
			Name:  src.Name,
			Type:  src.Type,
			Path:  src.Path,
			State: source.StateWaiting,
		})
	}

	ctx, cancel := context.WithCancel(parent)
	lineCh := make(chan source.LineEvent, 256)

	e.mu.Lock()
	e.cfg = cfg
	e.cancel = cancel
	e.statuses = initial
	e.opener.Command = cfg.Editor.Command
	e.notifier.Enabled = cfg.Notifications.Enabled
	e.notifier.Sound = cfg.Notifications.Sound
	e.running = true
	e.notifyCalls.Store(0)
	marks := e.marks
	e.mu.Unlock()

	followers := make([]*source.Follower, 0, len(cfg.Sources))
	for _, src := range cfg.Sources {
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
			OnStatus: func(s source.Status) {
				e.noteStatus(s)
				select {
				case e.statusCh <- s:
				default:
				}
			},
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

	e.wg.Add(1)
	go e.ingestLoop(ctx, parsers, lineCh)
	return nil
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

func (e *Engine) ingestLoop(ctx context.Context, parsers map[string]parser.Parser, lineCh <-chan source.LineEvent) {
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
