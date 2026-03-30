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
	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
)

const shutdownTimeout = 10 * time.Second

// HTTPCommand represents a command for running HTTP Server.
type HTTPCommand struct {
	depResolver depresolver.Resolver
	cfg         *config.Config
	logger      *slog.Logger

	address string
	port    string
}

// NewHTTPCommand creates a new HTTPCommand.
func NewHTTPCommand(depResolver depresolver.Resolver) *HTTPCommand {
	return &HTTPCommand{
		depResolver: depResolver,
		cfg:         depResolver.Config(),
		logger: depResolver.Logger().With(
			slog.String("component", "command"),
			slog.String("command", "serve http"),
		),
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

	server, err := c.depResolver.HTTPServer()
	if err != nil {
		return fmt.Errorf("resolving HTTP server: %w", err)
	}

	telegramUpdatesHandler, err := c.depResolver.TelegramUpdatesHandler()
	if err != nil {
		return fmt.Errorf("resolving Telegram updates handler: %w", err)
	}

	errGroup.Go(func() error {
		c.logger.InfoContext(ctx, "starting HTTP server",
			slog.String("address", c.cfg.Server.ListenAddr),
			slog.String("port", c.cfg.Server.Port),
		)

		// @todo: listen TLS if configured.
		err = server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listening and serving: %w", err)
		}

		return nil
	})

	errGroup.Go(func() error {
		c.logger.InfoContext(
			ctx,
			"starting telegram updates handler",
			slog.String("webhook", c.cfg.Telegram.Webhook.Path),
		)

		err = telegramUpdatesHandler.Handle(ctx)
		if err != nil {
			return fmt.Errorf("handling telegram updates: %w", err)
		}

		return nil
	})

	go func() {
		<-ctx.Done()
		c.logger.Info("termination signal received")

		c.shutdownStep(ctx, "stopping telegram updates handler", telegramUpdatesHandler.Stop)
		c.shutdownStep(ctx, "shutting down HTTP server", server.Shutdown)
	}()

	err = errGroup.Wait()
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
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
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
