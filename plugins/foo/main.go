// The example foo plugin.
package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/foo/db"
)

type FooPlugin struct {
	host   pluginapi.Host
	config *Config
	logger *slog.Logger
}

var (
	_ pluginapi.Plugin          = &FooPlugin{}
	_ pluginapi.CommandPlugin   = &FooPlugin{}
	_ pluginapi.CronPlugin      = &FooPlugin{}
	_ pluginapi.MigrationPlugin = &FooPlugin{}
)

// New creates a plugin instance.
//
//nolint:gochecknoglobals
var New pluginapi.Constructor = func(host pluginapi.Host, cfg map[string]any) (pluginapi.Plugin, error) {
	pluginConfig := new(Config)

	if err := pluginconfig.DecodeAndValidateConfig(cfg, pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	plg := &FooPlugin{
		host:   host,
		config: pluginConfig,
	}

	plg.logger = host.Logger().With(
		slog.String("component", "plugin"),
		slog.String("plugin", plg.Meta().ID.String()),
		slog.String("plugin-version", plg.Meta().Version),
		slog.String("plugin-api-version", plg.Meta().APIVersion),
	)

	return plg, nil
}

func (p *FooPlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.NewPluginIDFromString("dev.maroid.foo"),
		Version:    "0.1.0",
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *FooPlugin) RegisterCommands() []*cobra.Command {
	pluginCmd := &cobra.Command{
		Use:   "bar",
		Short: "Do bar",
		Run: func(_ *cobra.Command, _ []string) {
			p.logger.Info("Do bar stuff", slog.String("api-key", p.config.APIKey))
		},
	}

	return []*cobra.Command{
		pluginCmd,
	}
}

func (p *FooPlugin) RegisterCrons(scheduler *cron.Cron) error {
	_, err := scheduler.AddFunc("*/10 * * * * *", func() {
		p.logger.Info("job 1 has executed", slog.Time("time", time.Now()))
	})
	if err != nil {
		return fmt.Errorf("failed to register job: %w", err)
	}

	_, err = scheduler.AddFunc("*/20 * * * * *", func() {
		p.logger.Info("job 2 has executed", slog.Time("time", time.Now()))
	})
	if err != nil {
		return fmt.Errorf("failed to register job: %w", err)
	}

	return nil
}

func (p *FooPlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to get migration FS: %w", err)
	}

	return migrationsFS, nil
}
