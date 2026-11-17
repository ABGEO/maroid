package depresolver

import "github.com/abgeo/maroid/apps/hub/internal/registry"

// FileKindRegistry initializes and returns the registry of the file kinds of the plugins.
func (c *Container) FileKindRegistry() *registry.FileKindRegistry {
	c.fileKindRegistry.once.Do(func() {
		c.fileKindRegistry.instance = registry.NewFileKindRegistry()
	})

	return c.fileKindRegistry.instance
}
