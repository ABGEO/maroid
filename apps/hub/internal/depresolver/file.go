package depresolver

import (
	"github.com/abgeo/maroid/apps/hub/internal/filestore"
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
