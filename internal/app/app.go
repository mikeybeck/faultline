package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mikey/faultline/internal/config"
	"github.com/mikey/faultline/internal/editor"
	"github.com/mikey/faultline/internal/event"
	"github.com/mikey/faultline/internal/notify"
	"github.com/mikey/faultline/internal/parser"
	"github.com/mikey/faultline/internal/source"
	"github.com/mikey/faultline/internal/store"
	"github.com/mikey/faultline/internal/tui"
)

// Run starts watchers and the TUI.
func Run(ctx context.Context, cfg *config.Config, fromStart bool) error {
	st := store.New()
	notifier := &notify.Notifier{
		Enabled: cfg.Notifications.Enabled,
		Sound:   cfg.Notifications.Sound,
	}
	opener := editor.Opener{Command: cfg.Editor.Command}

	statusCh := make(chan source.Status, 64)
	lineCh := make(chan source.LineEvent, 256)
	eventCh := make(chan event.Event, 64)

	parsers := make(map[string]parser.Parser, len(cfg.Sources))
	initialStatuses := make([]source.Status, 0, len(cfg.Sources))
	for _, src := range cfg.Sources {
		p, err := parser.ForType(src.Type, src.Name)
		if err != nil {
			return err
		}
		parsers[src.Name] = p
		initialStatuses = append(initialStatuses, source.Status{
			Name:  src.Name,
			Type:  src.Type,
			Path:  src.Path,
			State: source.StateWaiting,
		})
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for _, src := range cfg.Sources {
		src := src
		wg.Add(1)
		go func() {
			defer wg.Done()
			f := &source.Follower{
				Name:      src.Name,
				Type:      src.Type,
				Path:      src.Path,
				FromStart: fromStart,
				OnStatus: func(s source.Status) {
					select {
					case statusCh <- s:
					default:
					}
				},
			}
			_ = f.Run(ctx, lineCh)
		}()
	}

	ingest := func(ev *event.Event) {
		res := st.Ingest(*ev)
		if res.IsNew {
			notifier.NewError(res.Event)
		}
		select {
		case eventCh <- res.Event:
		default:
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		flushTicker := time.NewTicker(250 * time.Millisecond)
		defer flushTicker.Stop()
		var lastActivity time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case <-flushTicker.C:
				// Emit pending multi-line records only after the stream goes idle.
				if !lastActivity.IsZero() && time.Since(lastActivity) >= 750*time.Millisecond {
					for _, p := range parsers {
						for _, ev := range p.Flush() {
							ingest(ev)
						}
					}
					lastActivity = time.Time{}
				}
			case line, ok := <-lineCh:
				if !ok {
					return
				}
				lastActivity = time.Now()
				p := parsers[line.Source]
				if p == nil {
					continue
				}
				for _, ev := range p.Feed(line.Line) {
					ingest(ev)
				}
			}
		}
	}()

	model := tui.New(tui.Deps{
		Store:         st,
		Open:          opener.Open,
		InitialStatus: initialStatuses,
		Events:        eventCh,
		StatusCh:      statusCh,
	})

	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := program.Run()
	cancel()

	for _, p := range parsers {
		for _, ev := range p.Flush() {
			st.Ingest(*ev)
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	default:
		// Don't hang forever if followers are mid-sleep.
	}

	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}
