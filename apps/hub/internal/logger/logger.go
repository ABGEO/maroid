// Package logger provides initialization and configuration for the structured
// slog logger used in the application.
package logger

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

// New creates and returns a new slog.Logger configured according to the
// provided configuration.
func New(cfg *config.Config) (*slog.Logger, error) {
	level, err := parseLogLevel(cfg.Logger.Level)
	if err != nil {
		return nil, err
	}

	handler := getDefaultHandler(level, cfg.Logger.Format, cfg.Env)

	return slog.New(handler), nil
}

func parseLogLevel(rawLevel string) (slog.Level, error) {
	var level slog.Level

	err := level.UnmarshalText([]byte(rawLevel))
	if err != nil {
		return level, fmt.Errorf("failed to parse log level: %w", err)
	}

	return level, nil
}

func getDefaultHandler(level slog.Level, format string, env string) slog.Handler {
	commonOptions := &slog.HandlerOptions{
		Level:     level,
		AddSource: env == "dev",
	}

	switch format {
	case "text":
		return slog.NewTextHandler(os.Stdout, commonOptions)
	case "json":
		return slog.NewJSONHandler(os.Stdout, commonOptions)
	default:
		return slog.NewJSONHandler(os.Stdout, commonOptions)
	}
}
