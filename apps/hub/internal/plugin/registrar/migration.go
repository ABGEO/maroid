package registrar

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// MigrationRegistrar is responsible for registering plugin migrations.
type MigrationRegistrar struct {
	registry *registry.MigrationRegistry
}

var _ Registrar = (*MigrationRegistrar)(nil)

// NewMigrationRegistrar creates a new MigrationRegistrar.
func NewMigrationRegistrar(reg *registry.MigrationRegistry) *MigrationRegistrar {
	return &MigrationRegistrar{
		registry: reg,
	}
}

// Name returns the name of the registrar.
func (r *MigrationRegistrar) Name() string {
	return "migration"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *MigrationRegistrar) Supports(plugin pluginapi.Plugin) bool {
	_, ok := plugin.(pluginapi.MigrationPlugin)

	return ok
}

// Register handles the registration of a plugin capability.
func (r *MigrationRegistrar) Register(plugin pluginapi.Plugin) error {
	id := plugin.Meta().ID

	migrationPlugin, ok := plugin.(pluginapi.MigrationPlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support Migration capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	migrations, err := migrationPlugin.Migrations()
	if err != nil {
		return fmt.Errorf("failed to retrieve migrations for plugin %s: %w", id, err)
	}

	err = r.registry.Register(plugin.Meta().ID.String(), migrations)
	if err != nil {
		return fmt.Errorf("failed to register migrations for plugin %s: %w", id, err)
	}

	return nil
}
