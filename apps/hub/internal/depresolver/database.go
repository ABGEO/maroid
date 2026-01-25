package depresolver

import (
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/migrator"
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

		return nil, fmt.Errorf("failed to initialize database: %w", err)
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
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	return nil
}

// Migrator initializes and returns the database migrator instance.
func (c *Container) Migrator() (*migrator.Migrator, error) {
	c.migrator.mu.Lock()
	defer c.migrator.mu.Unlock()

	var err error

	c.migrator.once.Do(func() {
		db, dbErr := c.Database()
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
			db,
			migrationRegistry,
		)
	})

	if err != nil {
		c.migrator.once = sync.Once{}

		return nil, fmt.Errorf("failed to initialize database migrator: %w", err)
	}

	return c.migrator.instance, nil
}
