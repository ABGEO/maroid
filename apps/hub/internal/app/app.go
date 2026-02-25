// Package app contains the main application logic for the hub.
package app

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/command"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
	pluginloader "github.com/abgeo/maroid/apps/hub/internal/plugin/loader"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// Application represents the main application.
type Application struct {
	resolver        depresolver.Resolver
	pluginLoader    *pluginloader.Loader
	commandRegistry *registry.CommandRegistry
	cfg             *config.Config
}

// New creates a new Application.
func New() (*Application, error) {
	depResolver, err := depresolver.NewResolver()
	if err != nil {
		return nil, fmt.Errorf("initializing dependency resolver: %w", err)
	}

	pluginLoader, err := depResolver.PluginLoader()
	if err != nil {
		return nil, fmt.Errorf("resolving plugin loader: %w", err)
	}

	commandRegistry, err := depResolver.CommandRegistry()
	if err != nil {
		return nil, fmt.Errorf("resolving command registry: %w", err)
	}

	return &Application{
		resolver:        depResolver,
		pluginLoader:    pluginLoader,
		commandRegistry: commandRegistry,
		cfg:             depResolver.Config(),
	}, nil
}

// Run executes the application with the given context.
func (a *Application) Run(ctx context.Context) error {
	if err := a.loadPlugins(); err != nil {
		return err
	}

	rootCmd := command.New().Command()
	rootCmd.PersistentPostRunE = func(_ *cobra.Command, _ []string) error {
		return a.cleanup(context.WithoutCancel(ctx))
	}

	rootCmd.AddCommand(a.commandRegistry.All()...)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		return fmt.Errorf("executing root command: %w", err)
	}

	return nil
}

func (a *Application) loadPlugins() error {
	for _, pluginCfg := range a.cfg.Plugins {
		if !pluginCfg.Enabled {
			continue
		}

		if err := a.pluginLoader.Load(pluginCfg.Path, pluginCfg.Config); err != nil {
			return fmt.Errorf("loading plugin %s: %w", pluginCfg.Path, err)
		}
	}

	return nil
}

func (a *Application) cleanup(ctx context.Context) error {
	const timeout = 10 * time.Second

	cleanupCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := a.resolver.Close(cleanupCtx); err != nil {
		return fmt.Errorf("closing dependencies: %w", err)
	}

	return nil
}
