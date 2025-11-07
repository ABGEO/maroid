// Package main contains the entry point for the Maroid Hub application.
package main

import (
	"log/slog"
	"os"

	"github.com/abgeo/maroid/apps/hub/internal/command"
)

func main() {
	err := command.Execute()
	if err != nil {
		slog.Error("failed to execute command", slog.Any("error", err))
		os.Exit(1)
	}
}
