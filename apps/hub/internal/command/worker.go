package command

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/worker"
)

const workerShutdownTimeout = 10 * time.Second

// WorkerCommand represents the command that runs background workers.
type WorkerCommand struct {
	logger  *slog.Logger
	workers []worker.Worker

	selectedWorkers []string
}

// NewWorkerCommand creates a new WorkerCommand.
func NewWorkerCommand(logger *slog.Logger, workers []worker.Worker) *WorkerCommand {
	return &WorkerCommand{
		logger: logger.With(
			slog.String("component", "command"),
			slog.String("command", "worker"),
		),
		workers: workers,
	}
}

// Command initializes and returns the Cobra command.
func (c *WorkerCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Run background workers",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return c.prepare()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.run(cmd.Context())
		},
	}

	cmd.Flags().StringSliceVarP(
		&c.selectedWorkers,
		"workers",
		"w",
		[]string{"all"},
		"Worker types to run, comma-separated (e.g. --workers cron,mqtt) or 'all' to run all workers",
	)

	return cmd
}

func (c *WorkerCommand) prepare() error {
	if len(c.selectedWorkers) > 0 {
		filtered, err := c.filterWorkers(c.selectedWorkers)
		if err != nil {
			return err
		}

		c.workers = filtered
	}

	for _, w := range c.workers {
		c.logger.Info("preparing worker", slog.String("worker", w.Name()))

		if err := w.Prepare(); err != nil {
			return fmt.Errorf("preparing worker %s: %w", w.Name(), err)
		}
	}

	return nil
}

func (c *WorkerCommand) filterWorkers(names []string) ([]worker.Worker, error) {
	if slices.Contains(names, "all") {
		return c.workers, nil
	}

	result := make([]worker.Worker, 0, len(names))
	index := make(map[string]worker.Worker, len(c.workers))

	for _, wrk := range c.workers {
		index[wrk.Name()] = wrk
	}

	for _, name := range names {
		wrk, ok := index[name]
		if !ok {
			return nil, fmt.Errorf(
				"%w: %q (available: %s)",
				errs.ErrUnknownWorkerType,
				name,
				strings.Join(slices.Collect(maps.Keys(index)), ", "),
			)
		}

		result = append(result, wrk)
	}

	return result, nil
}

func (c *WorkerCommand) run(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	for _, wrk := range c.workers {
		c.logger.InfoContext(ctx, "starting worker", slog.String("worker", wrk.Name()))

		errGroup.Go(func() error {
			if err := wrk.Start(ctx); err != nil {
				return fmt.Errorf("starting worker %s: %w", wrk.Name(), err)
			}

			return nil
		})
	}

	go func() {
		<-ctx.Done()
		c.logger.Info("termination signal received, shutting down workers")

		shutdownCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			workerShutdownTimeout,
		)
		defer cancel()

		for _, wrk := range c.workers {
			if err := wrk.Stop(shutdownCtx); err != nil {
				c.logger.ErrorContext(
					shutdownCtx,
					"worker stop failed",
					slog.String("worker", wrk.Name()),
					slog.Any("error", err),
				)
			}
		}
	}()

	if err := errGroup.Wait(); err != nil {
		return fmt.Errorf("worker error: %w", err)
	}

	return nil
}
