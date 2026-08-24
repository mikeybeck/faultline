package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mikey/faultline/internal/app"
	"github.com/mikey/faultline/internal/config"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: faultline [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Local error inbox for any log file (terminal UI).\n\n")
		flag.PrintDefaults()
	}
	configPath := flag.String("config", "", "path to faultline.yaml (default: search cwd)")
	fromStart := flag.Bool("from-start", false, "read existing log content from the beginning")
	flag.Parse()

	path := *configPath
	if path == "" {
		found, err := config.Find("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "faultline: %v\n\nCopy faultline.example.yaml to faultline.yaml and adjust paths.\n", err)
			os.Exit(1)
		}
		path = found
	}

	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "faultline: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, *fromStart); err != nil {
		fmt.Fprintf(os.Stderr, "faultline: %v\n", err)
		os.Exit(1)
	}
}
