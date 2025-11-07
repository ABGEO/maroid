package depresolver

import (
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
)

// Database returns the database instance.
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
