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
// The hub registers one handler for each type of its own: the tasks of the files and
// the destruction of a key.
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

		keys := secret.NewDestroyHandler(lazyKeyDestroyer{resolve: c.SecretKeyDestroyer})
		if err = handlers.Register(keys); err != nil {
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

// lazyKeyDestroyer reaches OpenBao at the first destruction, so a store that is down
// when the worker starts fails the tasks of the keys alone, and they retry. The tasks
// of the files run as before.
type lazyKeyDestroyer struct {
	resolve func() (secret.KeyDestroyer, error)
}

func (l lazyKeyDestroyer) DestroyKey(ctx context.Context, key secret.Key) error {
	keys, err := l.resolve()
	if err != nil {
		return fmt.Errorf("reaching the secret store: %w", err)
	}

	if err = keys.DestroyKey(ctx, key); err != nil {
		return fmt.Errorf("destroying through the secret store: %w", err)
	}

	return nil
}
