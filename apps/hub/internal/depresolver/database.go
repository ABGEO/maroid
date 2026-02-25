package depresolver

import (
	"fmt"
	"io/fs"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/migrator"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// Database initializes and returns the database instance.
func (c *Container) Database() (*sqlx.DB, error) {
	c.database.mu.Lock()
	defer c.database.mu.Unlock()

	var err error

	c.database.once.Do(func() {
		c.database.instance, err = database.New(c.Config())
	})

	if err != nil {
		c.database.once = sync.Once{}

		return nil, fmt.Errorf("initializing database: %w", err)
	}

	return c.database.instance, nil
}

// CloseDatabase closes the database connection.
func (c *Container) CloseDatabase() error {
	if c.database.instance == nil {
		return nil
	}

	err := c.database.instance.Close()
	if err != nil {
		return fmt.Errorf("closing database connection: %w", err)
	}

	return nil
}

// MigrationRegistry initializes and returns the migration registry instance.
func (c *Container) MigrationRegistry() (*registry.MigrationRegistry, error) {
	c.migrationRegistry.mu.Lock()
	defer c.migrationRegistry.mu.Unlock()

	var err error

	c.migrationRegistry.once.Do(func() {
		c.migrationRegistry.instance = registry.NewMigrationRegistry()

		coreFS, fsErr := getCoreMigrationFS()
		if fsErr != nil {
			err = fsErr

			return
		}

		err = c.migrationRegistry.instance.Register(migrator.TargetCore, coreFS)
	})

	if err != nil {
		c.migrationRegistry.once = sync.Once{}

		return nil, fmt.Errorf("initializing migration registry: %w", err)
	}

	return c.migrationRegistry.instance, nil
}

// Migrator initializes and returns the database migrator instance.
func (c *Container) Migrator() (*migrator.Migrator, error) {
	c.migrator.mu.Lock()
	defer c.migrator.mu.Unlock()

	var err error

	c.migrator.once.Do(func() {
		dbInstance, dbErr := c.Database()
		if dbErr != nil {
			err = dbErr

			return
		}

		migrationRegistry, migrationRegistryErr := c.MigrationRegistry()
		if migrationRegistryErr != nil {
			err = migrationRegistryErr

			return
		}

		c.migrator.instance = migrator.New(
			c.Config(),
			c.Logger(),
			dbInstance,
			migrationRegistry,
		)
	})

	if err != nil {
		c.migrator.once = sync.Once{}

		return nil, fmt.Errorf("initializing database migrator: %w", err)
	}

	return c.migrator.instance, nil
}

func getCoreMigrationFS() (fs.FS, error) {
	coreFS, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	if err != nil {
		return nil, fmt.Errorf("accessing core migrations: %w", err)
	}

	return coreFS, nil
}
