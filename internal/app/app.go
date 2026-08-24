package app

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mikey/faultline/internal/config"
	"github.com/mikey/faultline/internal/engine"
	"github.com/mikey/faultline/internal/tui"
)

// Run starts watchers and the TUI.
func Run(ctx context.Context, cfg *config.Config, fromStart bool) error {
	eng := engine.New()
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
	})

	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := program.Run()
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}
