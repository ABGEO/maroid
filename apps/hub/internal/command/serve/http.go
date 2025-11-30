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

	"github.com/abgeo/maroid/apps/hub/internal/appctx"
	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
	"github.com/abgeo/maroid/apps/hub/internal/telegram"
)

const shutdownTimeout = 10 * time.Second

type httpFlags struct {
	address string
	port    string
}

// NewHTTPCmd returns a new Cobra command for running HTTP Server.
func NewHTTPCmd(appCtx *appctx.AppContext) *cobra.Command {
	flags := httpFlags{}

	cmd := &cobra.Command{
		Use:   "http",
		Short: "Run HTTP server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger := appCtx.DepResolver.Logger().With(
				slog.String("component", "command"),
				slog.String("command", "serve http"),
			)

			return startServices(cmd.Context(), appCtx.DepResolver, logger)
		},
	}

	cmd.Flags().StringVarP(&flags.address, "address", "a", "0.0.0.0", "Server address")
	cmd.Flags().StringVarP(&flags.port, "port", "p", "8080", "Server port")

	return cmd
}

func startServices(
	ctx context.Context,
	depResolver depresolver.Resolver,
	logger *slog.Logger,
) error {
	srv, err := depResolver.HTTPServer()
	if err != nil {
		return fmt.Errorf("failed to resolve HTTP Server: %w", err)
	}

	uh, err := depResolver.TelegramUpdatesHandler()
	if err != nil {
		return fmt.Errorf("failed to resolve telegram updates handler: %w", err)
	}

	cfg := depResolver.Config()
	errGroup, ctx := errgroup.WithContext(ctx)

	// @todo: register plugin-provided handler.
	// @todo: register plugin-provided Telegram commands.

	errGroup.Go(func() error {
		logger.InfoContext(ctx, "starting HTTP server",
			slog.String("address", cfg.Server.ListenAddr),
			slog.String("port", cfg.Server.Port),
		)

		// @todo: listen TLS if configured.
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("failed to listen and serve: %w", err)
		}

		return nil
	})

	errGroup.Go(func() error {
		logger.InfoContext(
			ctx,
			"starting telegram updates handler",
			slog.String("webhook", cfg.Telegram.Webhook.Path),
		)

		err := uh.Handle()
		if err != nil {
			return fmt.Errorf("failed to handle telegram updates: %w", err)
		}

		return nil
	})

	go func() {
		<-ctx.Done()
		logger.Info("termination signal received")
		shutdownServices(context.WithoutCancel(ctx), depResolver, logger, srv, uh)
	}()

	err = errGroup.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("services errored: %w", err)
	}

	return nil
}

func shutdownServices(
	ctx context.Context,
	depResolver depresolver.Resolver,
	logger *slog.Logger,
	srv *http.Server,
	uh telegram.UpdatesHandler,
) {
	shutdownStep(ctx, logger, "stopping telegram updates handler", uh.Stop)
	shutdownStep(ctx, logger, "shutting down HTTP server", srv.Shutdown)
	shutdownStep(ctx, logger, "closing dependencies", depResolver.Close)
}

func shutdownStep(
	ctx context.Context,
	logger *slog.Logger,
	title string,
	step func(ctx context.Context) error,
) {
	ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	logger.InfoContext(ctx, title)

	if err := step(ctx); err != nil {
		logger.ErrorContext(
			ctx,
			"shutdown step failed",
			slog.String("step", title),
			slog.Any("error", err),
		)
	}
}
