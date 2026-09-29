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
	"github.com/abgeo/maroid/apps/hub/internal/healthcheck"
)

const shutdownTimeout = 10 * time.Second

// shutdownStep is one step of the shutdown, in the reverse order of the start.
type shutdownStep struct {
	title string
	run   func(ctx context.Context) error
}

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

	healthService, err := c.depResolver.HealthService()
	if err != nil {
		return fmt.Errorf("resolving the health service: %w", err)
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

	go c.shutdown(ctx, []shutdownStep{
		{"draining the HTTP server", c.drain(healthService)},
		{"stopping telegram updates handler", telegramUpdatesHandler.Stop},
		{"shutting down HTTP server", server.Shutdown},
	})

	err = errGroup.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("services errored: %w", err)
	}

	return nil
}

// drain fails the readiness, then keeps serving for the drain period, so that the
// orchestrator stops the traffic before the listener closes.
func (c *HTTPCommand) drain(healthService *healthcheck.Service) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		healthService.Drain()

		timer := time.NewTimer(c.cfg.Server.DrainPeriod)
		defer timer.Stop()

		select {
		case <-timer.C:
		case <-ctx.Done():
		}

		return nil
	}
}

// shutdown waits for the cancellation, then runs each step in turn.
func (c *HTTPCommand) shutdown(ctx context.Context, steps []shutdownStep) {
	<-ctx.Done()
	c.logger.Info("termination signal received")

	for _, step := range steps {
		c.runShutdownStep(ctx, step)
	}
}

func (c *HTTPCommand) runShutdownStep(ctx context.Context, step shutdownStep) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	c.logger.InfoContext(ctx, step.title)

	if err := step.run(ctx); err != nil {
		c.logger.ErrorContext(
			ctx,
			"shutdown step failed",
			slog.String("step", step.title),
			slog.Any("error", err),
		)
	}
}
