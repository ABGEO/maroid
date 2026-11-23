package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/file"
	"github.com/abgeo/maroid/apps/hub/internal/outbox"
)

// OutboxWorker initializes and returns the worker that runs the tasks of the outbox,
// with the handler of each type of the hub.
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

		c.outboxWorker.instance = outbox.NewWorker(c.Logger(), db, handlers, c.Config().Outbox)
	})

	if err != nil {
		c.outboxWorker.once = sync.Once{}

		return nil, fmt.Errorf("initializing the outbox worker: %w", err)
	}

	return c.outboxWorker.instance, nil
}
