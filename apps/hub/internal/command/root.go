// Package command provides Cobra CLI commands for running application and its plugins.
package command

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/appctx"
	"github.com/abgeo/maroid/apps/hub/internal/command/migrate"
	"github.com/abgeo/maroid/apps/hub/internal/command/serve"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
	pluginloader "github.com/abgeo/maroid/apps/hub/internal/plugin/loader"
)

// NewRootCmd creates and returns the root Cobra command.
func NewRootCmd(appCtx *appctx.AppContext) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use: "maroid",
	}

	// @todo: use
	cmd.PersistentFlags().
		String("config", "", `config file (default "$HOME/.maroid/config.yaml")`)

	err := registerSubcommands(appCtx, cmd)
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

// Execute runs the root Cobra command and handles OS interrupt and termination
// signals.
func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	appCtx, err := createAppContext()
	if err != nil {
		return err
	}

	rootCmd, err := NewRootCmd(appCtx)
	if err != nil {
		return err
	}

	err = rootCmd.ExecuteContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute root command: %w", err)
	}

	return nil
}

func createAppContext() (*appctx.AppContext, error) {
	depResolver, err := depresolver.NewResolver()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dependency resolver: %w", err)
	}

	pluginLoader, err := depResolver.PluginLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize plugin loader: %w", err)
	}

	err = loadPlugins(
		depResolver.Config(),
		pluginLoader,
	)
	if err != nil {
		return nil, err
	}

	return &appctx.AppContext{
		DepResolver: depResolver,
	}, nil
}

func loadPlugins(cfg *config.Config, pluginLoader *pluginloader.Loader) error {
	for _, pluginCfg := range cfg.Plugins {
		if !pluginCfg.Enabled {
			continue
		}

		err := pluginLoader.Load(pluginCfg.Path, pluginCfg.Config)
		if err != nil {
			return fmt.Errorf("failed to load plugin %s: %w", pluginCfg.Path, err)
		}
	}

	return nil
}

func registerSubcommands(appCtx *appctx.AppContext, parentCmd *cobra.Command) error {
	commandsRegistry, err := appCtx.DepResolver.CommandRegistry()
	if err != nil {
		return fmt.Errorf("failed to get command registry: %w", err)
	}

	commands := []*cobra.Command{
		NewCronCmd(appCtx),
		migrate.NewCmd(appCtx),
		serve.NewCmd(appCtx),
	}
	commands = append(commands, commandsRegistry.All()...)
	parentCmd.AddCommand(commands...)

	return nil
}
