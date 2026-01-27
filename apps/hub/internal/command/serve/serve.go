// Package serve provides Cobra commands for running servers.
package serve

import (
	"log/slog"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/telegram"
)

// Command represents a command for running servers.
type Command struct {
	cfg                    *config.Config
	logger                 *slog.Logger
	server                 *http.Server
	telegramUpdatesHandler telegram.UpdatesHandler
}

// New creates a new Command.
func New(
	cfg *config.Config,
	logger *slog.Logger,
	server *http.Server,
	telegramUpdatesHandler telegram.UpdatesHandler,
) *Command {
	return &Command{
		cfg:                    cfg,
		logger:                 logger,
		server:                 server,
		telegramUpdatesHandler: telegramUpdatesHandler,
	}
}

// Command initializes and returns the Cobra command.
func (c *Command) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run servers",
	}

	httpCommand := NewHTTPCommand(
		c.cfg,
		c.logger,
		c.server,
		c.telegramUpdatesHandler,
	)

	cmd.AddCommand(
		httpCommand.Command(),
	)

	return cmd
}
