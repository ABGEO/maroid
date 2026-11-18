package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/file"
	"github.com/abgeo/maroid/apps/hub/internal/filestore"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// FileKindRegistry initializes and returns the registry of the file kinds of the plugins.
func (c *Container) FileKindRegistry() *registry.FileKindRegistry {
	c.fileKindRegistry.once.Do(func() {
		c.fileKindRegistry.instance = registry.NewFileKindRegistry()
	})

	return c.fileKindRegistry.instance
}

// FileStore initializes and returns the client of the bucket of the files.
func (c *Container) FileStore() *filestore.Client {
	c.fileStore.once.Do(func() {
		c.fileStore.instance = filestore.New(c.Config().Files)
	})

	return c.fileStore.instance
}

// Files initializes and returns the store of the files of every plugin.
func (c *Container) Files() (*file.Store, error) {
	c.files.mu.Lock()
	defer c.files.mu.Unlock()

	var err error

	c.files.once.Do(func() {
		db, dbErr := c.Database()
		if dbErr != nil {
			err = dbErr

			return
		}

		c.files.instance = file.NewStore(
			db,
			c.FileStore(),
			c.FileKindRegistry(),
			c.Config().Files.UploadTTL,
		)
	})

	if err != nil {
		c.files.once = sync.Once{}

		return nil, fmt.Errorf("initializing the file store: %w", err)
	}

	return c.files.instance, nil
}

// buildFileHandler resolves every dependency of the handler of the uploads.
func (c *Container) buildFileHandler() (*handler.File, error) {
	verifier, err := c.TokenVerifier()
	if err != nil {
		return nil, err
	}

	identityResolver, err := c.IdentityResolver()
	if err != nil {
		return nil, err
	}

	idempotencyStore, err := c.IdempotencyStore()
	if err != nil {
		return nil, err
	}

	access, err := c.workspaceAccess()
	if err != nil {
		return nil, err
	}

	files, err := c.Files()
	if err != nil {
		return nil, err
	}

	return handler.NewFile(
		c.Logger(),
		verifier,
		identityResolver,
		idempotencyStore,
		access,
		files,
	), nil
}

// FileWorker initializes and returns the worker that runs the tasks of the files.
func (c *Container) FileWorker() (*file.Worker, error) {
	c.fileWorker.mu.Lock()
	defer c.fileWorker.mu.Unlock()

	var err error

	c.fileWorker.once.Do(func() {
		db, dbErr := c.Database()
		if dbErr != nil {
			err = dbErr

			return
		}

		c.fileWorker.instance = file.NewWorker(
			c.Logger(),
			db,
			c.FileStore(),
			c.FileKindRegistry(),
			c.Config().Files,
		)
	})

	if err != nil {
		c.fileWorker.once = sync.Once{}

		return nil, fmt.Errorf("initializing the files worker: %w", err)
	}

	return c.fileWorker.instance, nil
}
