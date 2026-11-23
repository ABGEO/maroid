package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/worker"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	workerName = "outbox"
	// leaseLength holds a task away from another worker while one runs it. It passes
	// the longest processing of an image many times over.
	leaseLength = 5 * time.Minute
	// defaultTaskTimeout ends a task before its lease lapses, so no other worker takes
	// it while it still runs.
	defaultTaskTimeout = 4 * time.Minute
	retryBase          = 2 * time.Second
	retryLimit         = 5 * time.Minute
)

var (
	errNoHandler = errors.New("outbox: no handler serves the type")
	errPanicked  = errors.New("outbox: the task panicked")
)

// Option changes a Worker at its creation.
type Option func(*Worker)

// WithTaskTimeout replaces the deadline of each task.
func WithTaskTimeout(timeout time.Duration) Option {
	return func(w *Worker) { w.taskTimeout = timeout }
}

// Worker leases the due tasks of the outbox, and runs each one through the handler of
// its type, in the scope of its owner.
type Worker struct {
	logger       *slog.Logger
	db           *sqlx.DB
	handlers     *Registry
	pollInterval time.Duration
	concurrency  int
	taskTimeout  time.Duration
	cycles       sync.WaitGroup
}

var _ worker.Worker = (*Worker)(nil)

// NewWorker creates a Worker over the database and the handlers.
func NewWorker(
	logger *slog.Logger,
	db *sqlx.DB,
	handlers *Registry,
	cfg config.Outbox,
	options ...Option,
) *Worker {
	outboxWorker := &Worker{
		logger: logger.With(
			slog.String("component", "worker"),
			slog.String("worker", workerName),
		),
		db:           db,
		handlers:     handlers,
		pollInterval: cfg.PollInterval,
		concurrency:  cfg.Concurrency,
		taskTimeout:  defaultTaskTimeout,
	}

	for _, option := range options {
		option(outboxWorker)
	}

	return outboxWorker
}

// Name returns the worker type identifier.
func (w *Worker) Name() string { return workerName }

// Prepare runs the check of each handler that has one, so a broken dependency fails at
// the start and not at the first task.
func (w *Worker) Prepare() error {
	for _, handler := range w.handlers.All() {
		preparer, ok := handler.(Preparer)
		if !ok {
			continue
		}

		if err := preparer.Prepare(); err != nil {
			return fmt.Errorf("preparing the handler of %s: %w", handler.Type(), err)
		}
	}

	return nil
}

// Start runs a cycle at each poll interval until the context ends. A cycle that
// filled every slot runs the next one at once, since more tasks may wait.
func (w *Worker) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		full, err := w.cycle(ctx)
		if err != nil {
			w.logger.ErrorContext(ctx, "leasing the tasks", slog.Any("error", err))
		}

		if full && ctx.Err() == nil {
			continue
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// Stop waits for the running cycle to end, or for the deadline of the context.
func (w *Worker) Stop(ctx context.Context) error {
	ended := make(chan struct{})

	go func() {
		w.cycles.Wait()
		close(ended)
	}()

	select {
	case <-ended:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stopping the outbox worker: %w", ctx.Err())
	}
}

// Cycle leases at most the concurrency of due tasks, runs them at one time, and
// returns when each one ends. A task ends its work after a shutdown starts, so the
// lease of a task never lapses in the middle.
func (w *Worker) Cycle(ctx context.Context) error {
	_, err := w.cycle(ctx)

	return err
}

// cycle runs one Cycle, and reports whether the lease filled every slot.
func (w *Worker) cycle(ctx context.Context) (bool, error) {
	w.cycles.Add(1)
	defer w.cycles.Done()

	var leased []Task

	err := w.inTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		if leased, err = repository.NewOutboxTask(tx).
			Lease(ctx, w.concurrency, leaseLength); err != nil {
			return fmt.Errorf("leasing: %w", err)
		}

		return nil
	})
	if err != nil {
		return false, fmt.Errorf("leasing the tasks of the outbox: %w", err)
	}

	running := context.WithoutCancel(ctx)

	var tasks sync.WaitGroup

	for _, task := range leased {
		tasks.Go(func() { w.run(running, task) })
	}

	tasks.Wait()

	return len(leased) == w.concurrency, nil
}

// run performs one task, then removes it or releases it for a retry.
func (w *Worker) run(ctx context.Context, task Task) {
	logger := w.logger.With(
		slog.String("task_id", task.ID),
		slog.String("type", task.Type),
		slog.Int("attempts", task.Attempts),
	)

	err := w.attempt(ctx, logger, task)
	if err == nil {
		if doneErr := w.inTx(ctx, func(tx *sqlx.Tx) error {
			return repository.NewOutboxTask(tx).Done(ctx, task.ID)
		}); doneErr != nil {
			logger.ErrorContext(ctx, "removing a done task", slog.Any("error", doneErr))
		}

		return
	}

	delay := retryDelay(task.Attempts)
	logger.WarnContext(
		ctx,
		"a task of the outbox failed",
		slog.Any("error", err),
		slog.Duration("retry_in", delay),
	)

	if retryErr := w.inTx(ctx, func(tx *sqlx.Tx) error {
		return repository.NewOutboxTask(tx).Retry(ctx, task.ID, delay, err.Error())
	}); retryErr != nil {
		logger.ErrorContext(ctx, "releasing a failed task", slog.Any("error", retryErr))
	}
}

// attempt calls the handler of the task within its deadline, as the owner of the task.
// A panic of the handler fails the attempt and leaves the worker running.
func (w *Worker) attempt(ctx context.Context, logger *slog.Logger, task Task) (err error) {
	handler, found := w.handlers.Get(task.Type)
	if !found {
		return fmt.Errorf("%w: %s", errNoHandler, task.Type)
	}

	bounded, cancel := context.WithTimeout(scopeOf(ctx, task), w.taskTimeout)
	defer cancel()

	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}

		logger.ErrorContext(
			ctx,
			"a task of the outbox panicked",
			slog.Any("panic", recovered),
			slog.String("stack", string(debug.Stack())),
		)

		err = fmt.Errorf("%w: %v", errPanicked, recovered)
	}()

	if err = handler.Handle(bounded, task); err != nil {
		return fmt.Errorf("handling %s: %w", task.Type, err)
	}

	return nil
}

// scopeOf answers a context that acts as the owner of the task, so the policy of a
// scoped table shows the rows of the owner.
func scopeOf(ctx context.Context, task Task) context.Context {
	owner, found := task.Owner()
	if !found {
		return ctx
	}

	if owner.Scope == model.OutboxScopeUser {
		return pluginapi.ContextWithActingUser(ctx, owner.ID)
	}

	return pluginapi.ContextWithActingWorkspace(ctx, owner.ID)
}

// inTx runs fn in a transaction with no scope, as the shared table of the tasks needs.
func (w *Worker) inTx(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := w.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning a transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing a transaction: %w", err)
	}

	return nil
}

// retryDelay answers 2 seconds after the first attempt, and twice as long after each
// later one, up to 5 minutes. The lease counts an attempt before the task runs, so the
// first failure holds 1.
func retryDelay(attempts int) time.Duration {
	delay := retryBase

	for range attempts - 1 {
		delay *= 2
		if delay >= retryLimit {
			return retryLimit
		}
	}

	return delay
}
