// Package command provides Cobra CLI commands for running application and its plugins.
package command

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/loader"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// AppContext holds application-wide dependencies.
type AppContext struct {
	DepResolver depresolver.Resolver
	PluginHost  pluginapi.Host
	Plugins     []pluginapi.Plugin
}

// NewRootCmd creates and returns the root Cobra command.
func NewRootCmd(appCtx *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use: "maroid",
	}

	// @todo: use
	cmd.PersistentFlags().
		String("config", "", `config file (default "$HOME/.maroid/config.yaml")`)

	registerSubcommands(appCtx, cmd)

	return cmd
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

	rootCmd := NewRootCmd(appCtx)

	err = rootCmd.ExecuteContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute root command: %w", err)
	}

	return nil
}

func createAppContext() (*AppContext, error) {
	depResolver, err := depresolver.NewResolver()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dependency resolver: %w", err)
	}

	pluginHost, err := host.New(depResolver)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize plugin host: %w", err)
	}

	plugins, err := loadPlugins(
		depResolver.Config(),
		pluginHost,
	)
	if err != nil {
		return nil, err
	}

	return &AppContext{
		DepResolver: depResolver,
		PluginHost:  pluginHost,
		Plugins:     plugins,
	}, nil
}

func loadPlugins(
	cfg *config.Config,
	pluginHost pluginapi.Host,
) ([]pluginapi.Plugin, error) {
	plugins := make([]pluginapi.Plugin, 0, len(cfg.Plugins))

	for _, pluginCfg := range cfg.Plugins {
		if !pluginCfg.Enabled {
			continue
		}

		plg, err := loader.LoadPlugin(pluginCfg.Path, pluginHost, pluginCfg.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to load plugin %s: %w", pluginCfg.Path, err)
		}

		plugins = append(plugins, plg)
	}

	return plugins, nil
}

func registerSubcommands(appCtx *AppContext, parentCmd *cobra.Command) {
	commands := []*cobra.Command{
		NewCronCmd(appCtx),
	}
	commands = append(commands, getPluginCommands(appCtx.Plugins)...)
	parentCmd.AddCommand(commands...)
}

func getPluginCommands(plugins []pluginapi.Plugin) []*cobra.Command {
	commands := make([]*cobra.Command, 0, len(plugins))

	for _, plg := range plugins {
		cmdPlugin, ok := plg.(pluginapi.CommandPlugin)
		if !ok {
			continue
		}

		pluginCommands := cmdPlugin.RegisterCommands()
		if len(pluginCommands) == 0 {
			continue
		}

		meta := plg.Meta()
		id := pluginapi.ParsePluginID(meta.ID)

		cmd := &cobra.Command{
			Use:   id.Name,
			Short: "Commands provided by plugin " + meta.ID,
			Long: fmt.Sprintf(
				"Commands registered by plugin %s (version: %s).",
				meta.ID, meta.Version,
			),
		}

		cmd.AddCommand(pluginCommands...)
		commands = append(commands, cmd)
	}

	return commands
}
