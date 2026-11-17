package file

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/gen2brain/h265/heic"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/filestore"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/worker"
)

const (
	workerName = "files"
	// leaseLength holds a task away from another worker while one runs it. It passes
	// the longest processing of an image many times over.
	leaseLength = 5 * time.Minute
	retryBase   = 2 * time.Second
	retryLimit  = 5 * time.Minute
	fixtureEdge = 16
)

// Objects is the part of the file store that the worker calls.
type Objects interface {
	Ping(ctx context.Context) error
	Head(ctx context.Context, key string) (filestore.Object, error)
	ReadRange(ctx context.Context, key string, limit int64) ([]byte, error)
	Read(ctx context.Context, key string) (io.ReadCloser, error)
	Write(ctx context.Context, key string, data []byte, contentType string) error
	Delete(ctx context.Context, key string) error
	DeletePrefix(ctx context.Context, prefix string) error
}

// Worker runs the due tasks of the files: it processes an upload, checks an upload
// that no completion reported, removes an upload that the variants replace, expires an
// unclaimed file, and purges the bytes of a deleted one.
type Worker struct {
	logger       *slog.Logger
	db           *sqlx.DB
	objects      Objects
	kinds        *registry.FileKindRegistry
	pollInterval time.Duration
	concurrency  int
	cycles       sync.WaitGroup
}

var _ worker.Worker = (*Worker)(nil)

// NewWorker creates a Worker over the database, the file store, and the file kinds.
func NewWorker(
	logger *slog.Logger,
	db *sqlx.DB,
	objects Objects,
	kinds *registry.FileKindRegistry,
	cfg config.Files,
) *Worker {
	return &Worker{
		logger: logger.With(
			slog.String("component", "worker"),
			slog.String("worker", workerName),
		),
		db:           db,
		objects:      objects,
		kinds:        kinds,
		pollInterval: cfg.PollInterval,
		concurrency:  cfg.Concurrency,
	}
}

// Name returns the worker type identifier.
func (w *Worker) Name() string { return workerName }

// Prepare checks that the file store answers and that each codec decodes and encodes
// an image, so a broken build fails at the start and not at the first upload.
func (w *Worker) Prepare() error {
	if err := w.objects.Ping(context.Background()); err != nil {
		return fmt.Errorf("preparing the files worker: %w", err)
	}

	if err := checkCodecs(); err != nil {
		return fmt.Errorf("preparing the files worker: %w", err)
	}

	return nil
}

// Start runs a cycle at each poll interval until the context ends.
func (w *Worker) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		if err := w.Cycle(ctx); err != nil {
			w.logger.ErrorContext(ctx, "leasing the tasks of the files", slog.Any("error", err))
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
		return fmt.Errorf("stopping the files worker: %w", ctx.Err())
	}
}

// Cycle leases at most the concurrency of due tasks, runs them at one time, and
// returns when each one ends. A task ends its work after a shutdown starts, so the
// lease of a task never lapses in the middle.
func (w *Worker) Cycle(ctx context.Context) error {
	w.cycles.Add(1)
	defer w.cycles.Done()

	var leased []model.FileTask

	err := w.inTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		if leased, err = repository.NewFileTask(tx).
			Lease(ctx, w.concurrency, leaseLength); err != nil {
			return fmt.Errorf("leasing: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("leasing the tasks of the files: %w", err)
	}

	running := context.WithoutCancel(ctx)

	var tasks sync.WaitGroup

	for _, task := range leased {
		tasks.Go(func() { w.run(running, task) })
	}

	tasks.Wait()

	return nil
}

// run performs one task, then removes it or releases it for a retry.
func (w *Worker) run(ctx context.Context, task model.FileTask) {
	logger := w.logger.With(
		slog.String("task_id", task.ID),
		slog.String("action", string(task.Action)),
		slog.String("file_id", task.FileID),
		slog.Int("attempts", task.Attempts),
	)

	err := w.perform(ctx, task)
	if err == nil {
		if doneErr := w.inTx(ctx, func(tx *sqlx.Tx) error {
			return repository.NewFileTask(tx).Done(ctx, task.ID)
		}); doneErr != nil {
			logger.ErrorContext(ctx, "removing a done task", slog.Any("error", doneErr))
		}

		return
	}

	delay := retryDelay(task.Attempts)
	logger.WarnContext(
		ctx,
		"a task of a file failed",
		slog.Any("error", err),
		slog.Duration("retry_in", delay),
	)

	if retryErr := w.inTx(ctx, func(tx *sqlx.Tx) error {
		return repository.NewFileTask(tx).
			Retry(ctx, task.ID, delay, err.Error())
	}); retryErr != nil {
		logger.ErrorContext(ctx, "releasing a failed task", slog.Any("error", retryErr))
	}
}

func (w *Worker) perform(ctx context.Context, task model.FileTask) error {
	scoped := scopeOf(ctx, task)

	switch task.Action {
	case model.FileTaskProcess:
		return w.process(scoped, task)
	case model.FileTaskCheck:
		return w.check(scoped, task)
	case model.FileTaskDiscardUpload:
		return w.discardUpload(scoped, task)
	case model.FileTaskExpire:
		return w.expire(scoped, task)
	case model.FileTaskPurge:
		return w.purge(scoped, task)
	default:
		return fmt.Errorf("%w: %s", errUnknownAction, task.Action)
	}
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

// retryDelay doubles from 2 seconds with each attempt, up to 5 minutes.
func retryDelay(attempts int) time.Duration {
	delay := retryBase

	for range attempts {
		delay *= 2
		if delay >= retryLimit {
			return retryLimit
		}
	}

	return delay
}

// checkCodecs runs a small PNG and a small HEIC through the pipeline.
func checkCodecs() error {
	fixture := image.NewNRGBA(image.Rect(0, 0, fixtureEdge, fixtureEdge))

	var pngBytes, heicBytes bytes.Buffer

	if err := png.Encode(&pngBytes, fixture); err != nil {
		return fmt.Errorf("encoding the PNG fixture: %w", err)
	}

	if err := heic.Encode(&heicBytes, fixture); err != nil {
		return fmt.Errorf("encoding the HEIC fixture: %w", err)
	}

	for mediaType, data := range map[string][]byte{pngType: pngBytes.Bytes(), heicType: heicBytes.Bytes()} {
		if _, err := variantsOf(data, mediaType); err != nil {
			return fmt.Errorf("deriving the variants of the %s fixture: %w", mediaType, err)
		}
	}

	return nil
}

var errUnknownAction = errors.New("file: unknown task action")
