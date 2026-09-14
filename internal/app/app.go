package app

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mikey/faultline/internal/config"
	"github.com/mikey/faultline/internal/engine"
	"github.com/mikey/faultline/internal/mark"
	"github.com/mikey/faultline/internal/persist"
	"github.com/mikey/faultline/internal/tui"
)

// Run starts watchers and the TUI.
func Run(ctx context.Context, cfg *config.Config, fromStart bool) error {
	eng := engine.New()
	eng.UseMarks(mark.Default())
	if db, err := persist.Default(); err == nil {
		eng.UsePersist(db)
		defer eng.ClosePersist()
	}
	if wd, err := os.Getwd(); err == nil {
		eng.SetProjectDir(wd)
	}
	if err := eng.Start(ctx, cfg, fromStart); err != nil {
		return err
	}
	defer eng.Stop()

	model := tui.New(tui.Deps{
		Store:         eng.Store(),
		Open:          eng.Open,
		InitialStatus: eng.SnapshotStatuses(),
		Events:        eng.Events(),
		StatusCh:      eng.Statuses(),
		Clear:         eng.Clear,
		Dismiss:       eng.Dismiss,
		ResetMark: func() {
			eng.ClearMarks()
			_ = eng.Restart(ctx, cfg, fromStart)
		},
	})

	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := program.Run()
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}
