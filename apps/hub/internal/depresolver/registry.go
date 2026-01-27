package depresolver

import (
	"fmt"
	"sync"

	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/command"
	migratecommand "github.com/abgeo/maroid/apps/hub/internal/command/migrate"
	servecommand "github.com/abgeo/maroid/apps/hub/internal/command/serve"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// CommandRegistry initializes and returns the command registry instance.
func (c *Container) CommandRegistry() (*registry.CommandRegistry, error) {
	c.commandRegistry.mu.Lock()
	defer c.commandRegistry.mu.Unlock()

	var err error

	c.commandRegistry.once.Do(func() {
		c.commandRegistry.instance = registry.NewCommandRegistry()

		commands, cmdErr := c.getCommands()
		if err != nil {
			err = cmdErr
		}

		err = c.commandRegistry.instance.Register(commands...)
	})

	if err != nil {
		c.commandRegistry.once = sync.Once{}

		return nil, fmt.Errorf("failed to initialize command registry: %w", err)
	}

	return c.commandRegistry.instance, nil
}

func (c *Container) getCommands() ([]*cobra.Command, error) {
	cfg := c.Config()
	logger := c.Logger()
	cron := c.Cron()

	cronRegistry, err := c.CronRegistry()
	if err != nil {
		return nil, err
	}

	migrator, err := c.Migrator()
	if err != nil {
		return nil, err
	}

	httpServer, err := c.HTTPServer()
	if err != nil {
		return nil, err
	}

	telegramUpdatesHandler, err := c.TelegramUpdatesHandler()
	if err != nil {
		return nil, err
	}

	cronCmd := command.NewCronCommand(logger, cron, cronRegistry)
	migrateCmd := migratecommand.New(migrator)
	serveCmd := servecommand.New(
		cfg,
		logger,
		httpServer,
		telegramUpdatesHandler,
	)

	return []*cobra.Command{
		cronCmd.Command(),
		migrateCmd.Command(),
		serveCmd.Command(),
	}, nil
}
