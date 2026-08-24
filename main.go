package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"

	"github.com/mikey/faultline/internal/app"
	"github.com/mikey/faultline/internal/config"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: faultline [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Local error inbox for any log file.\n\n")
		flag.PrintDefaults()
	}
	useTUI := flag.Bool("tui", false, "run the terminal UI instead of the desktop app")
	configPath := flag.String("config", "", "path to faultline.yaml (default: last project, or search cwd for --tui)")
	fromStart := flag.Bool("from-start", true, "ingest existing log content, then follow new lines")
	tailOnly := flag.Bool("tail", false, "follow new lines only; skip content already in the file")
	flag.Parse()

	ingestFromStart := *fromStart && !*tailOnly

	if *useTUI {
		if err := runTUI(*configPath, ingestFromStart); err != nil {
			fmt.Fprintf(os.Stderr, "faultline: %v\n", err)
			os.Exit(1)
		}
		return
	}

	gui := NewApp(*configPath, ingestFromStart)
	err := wails.Run(&options.App{
		Title:     "Faultline",
		Width:     1200,
		Height:    800,
		MinWidth:  860,
		MinHeight: 560,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 20, B: 26, A: 255},
		OnStartup:        gui.startup,
		OnShutdown:       gui.shutdown,
		Bind: []interface{}{
			gui,
		},
		Linux: &linux.Options{
			WebviewGpuPolicy: linux.WebviewGpuPolicyNever,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "faultline: %v\n", err)
		os.Exit(1)
	}
}

func runTUI(configPath string, fromStart bool) error {
	path := configPath
	if path == "" {
		found, err := config.Find("")
		if err != nil {
			return fmt.Errorf("%w\n\nCopy faultline.example.yaml to faultline.yaml and adjust paths, or run without --tui to use the desktop app.", err)
		}
		path = found
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return app.Run(ctx, cfg, fromStart)
}
