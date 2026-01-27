// Package main contains the entry point for the Maroid Hub application.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/abgeo/maroid/apps/hub/internal/app"
)

func main() {
	err := run()
	if err != nil {
		slog.Error("failed to run application", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application, err := app.New()
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	err = application.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run application: %w", err)
	}

	return nil
}
