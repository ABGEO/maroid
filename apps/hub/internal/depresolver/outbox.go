package depresolver

import (
	"context"
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/file"
	"github.com/abgeo/maroid/apps/hub/internal/outbox"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
)

// OutboxWorker initializes and returns the worker that runs the tasks of the outbox.
// The hub registers one handler for each type of its own: the tasks of the files, and
// the creation and the destruction of a key.
func (c *Container) OutboxWorker() (*outbox.Worker, error) {
	c.outboxWorker.mu.Lock()
	defer c.outboxWorker.mu.Unlock()

	var err error

	c.outboxWorker.once.Do(func() {
		db, dbErr := c.Database()
		if dbErr != nil {
			err = dbErr

			return
		}

		handlers := outbox.NewRegistry()

		files := file.NewHandlers(c.Logger(), db, c.FileStore(), c.FileKindRegistry())
		if err = handlers.Register(files.All()...); err != nil {
			return
		}

		keys := lazyKeys{resolve: c.SecretKeys}

		err = handlers.Register(
			secret.NewCreateHandler(c.Logger(), db, keys),
			secret.NewDestroyHandler(keys),
		)
		if err != nil {
			return
		}

		c.outboxWorker.instance = outbox.NewWorker(c.Logger(), db, handlers, c.Config().Outbox)
	})

	if err != nil {
		c.outboxWorker.once = sync.Once{}

		return nil, fmt.Errorf("initializing the outbox worker: %w", err)
	}

	return c.outboxWorker.instance, nil
}

// lazyKeys reaches OpenBao at the first task of a key, so a store that is down when the
// worker starts fails the tasks of the keys alone, and they retry. The tasks of the
// files run as before.
type lazyKeys struct {
	resolve func() (secret.Keys, error)
}

func (l lazyKeys) CreateKey(ctx context.Context, key secret.Key) error {
	keys, err := l.resolve()
	if err != nil {
		return fmt.Errorf("reaching the secret store: %w", err)
	}

	if err = keys.CreateKey(ctx, key); err != nil {
		return fmt.Errorf("creating through the secret store: %w", err)
	}

	return nil
}

func (l lazyKeys) DestroyKey(ctx context.Context, key secret.Key) error {
	keys, err := l.resolve()
	if err != nil {
		return fmt.Errorf("reaching the secret store: %w", err)
	}

	if err = keys.DestroyKey(ctx, key); err != nil {
		return fmt.Errorf("destroying through the secret store: %w", err)
	}

	return nil
}
