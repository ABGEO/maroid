// Package registrar provides functionality for registering and managing plugin capabilities.
package registrar

import "github.com/abgeo/maroid/apps/hub/internal/registry"

// Registrar defines the interface for plugin capability registrars.
type Registrar interface {
	// Name returns the name of the registrar.
	Name() string
	// Supports indicates whether the registrar can handle the given plugin.
	Supports(plugin registry.LoadedPlugin) bool
	// Register handles the registration of a plugin capability.
	Register(plugin registry.LoadedPlugin) error
}
