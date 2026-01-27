package serve

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/telegram"
)

const shutdownTimeout = 10 * time.Second

// HTTPCommand represents a command for running HTTP Server.
type HTTPCommand struct {
	cfg                    *config.Config
	logger                 *slog.Logger
	server                 *http.Server
	telegramUpdatesHandler telegram.UpdatesHandler

	address string
	port    string
}

// NewHTTPCommand creates a new HTTPCommand.
func NewHTTPCommand(
	cfg *config.Config,
	logger *slog.Logger,
	server *http.Server,
	telegramUpdatesHandler telegram.UpdatesHandler,
) *HTTPCommand {
	return &HTTPCommand{
		cfg: cfg,
		logger: logger.With(
			slog.String("component", "command"),
			slog.String("command", "serve http"),
		),
		server:                 server,
		telegramUpdatesHandler: telegramUpdatesHandler,
	}
}

// Command initializes and returns the Cobra command.
func (c *HTTPCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "http",
		Short: "Run HTTP server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.startServices(cmd.Context())
		},
	}

	// @todo: use
	cmd.Flags().StringVarP(&c.address, "address", "a", "0.0.0.0", "Server address")
	cmd.Flags().StringVarP(&c.port, "port", "p", "8080", "Server port")

	return cmd
}

func (c *HTTPCommand) startServices(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	// @todo: register plugin-provided handler.

	errGroup.Go(func() error {
		c.logger.InfoContext(ctx, "starting HTTP server",
			slog.String("address", c.cfg.Server.ListenAddr),
			slog.String("port", c.cfg.Server.Port),
		)

		// @todo: listen TLS if configured.
		err := c.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("failed to listen and serve: %w", err)
		}

		return nil
	})

	errGroup.Go(func() error {
		c.logger.InfoContext(
			ctx,
			"starting telegram updates handler",
			slog.String("webhook", c.cfg.Telegram.Webhook.Path),
		)

		err := c.telegramUpdatesHandler.Handle(ctx)
		if err != nil {
			return fmt.Errorf("failed to handle telegram updates: %w", err)
		}

		return nil
	})

	go func() {
		<-ctx.Done()
		c.logger.Info("termination signal received")

		c.shutdownStep(ctx, "stopping telegram updates handler", c.telegramUpdatesHandler.Stop)
		c.shutdownStep(ctx, "shutting down HTTP server", c.server.Shutdown)
	}()

	err := errGroup.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("services errored: %w", err)
	}

	return nil
}

func (c *HTTPCommand) shutdownStep(
	ctx context.Context,
	title string,
	step func(ctx context.Context) error,
) {
	ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	c.logger.InfoContext(ctx, title)

	if err := step(ctx); err != nil {
		c.logger.ErrorContext(
			ctx,
			"shutdown step failed",
			slog.String("step", title),
			slog.Any("error", err),
		)
	}
}
