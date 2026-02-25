package registry

import (
	"fmt"
	"io/fs"
	"maps"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// MigrationRegistry is a registry for database migrations.
type MigrationRegistry struct {
	migrations map[string]fs.FS
}

// NewMigrationRegistry creates a new MigrationRegistry.
func NewMigrationRegistry() *MigrationRegistry {
	return &MigrationRegistry{
		migrations: make(map[string]fs.FS),
	}
}

// Register registers a new migration source.
func (r *MigrationRegistry) Register(source string, migration fs.FS) error {
	if _, exists := r.migrations[source]; exists {
		return fmt.Errorf("%w: %s", errs.ErrMigrationSourceAlreadyRegistered, source)
	}

	r.migrations[source] = migration

	return nil
}

// All returns a copy of all registered migration sources.
func (r *MigrationRegistry) All() map[string]fs.FS {
	out := make(map[string]fs.FS, len(r.migrations))

	maps.Copy(out, r.migrations)

	return out
}
