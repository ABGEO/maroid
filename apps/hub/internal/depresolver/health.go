package depresolver

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/healthcheck"
)

// HealthService initializes and returns the service that measures the health of the hub.
func (c *Container) HealthService() (*healthcheck.Service, error) {
	c.healthService.mu.Lock()
	defer c.healthService.mu.Unlock()

	var err error

	c.healthService.once.Do(func() {
		db, dbErr := c.Database()
		if dbErr != nil {
			err = dbErr

			return
		}

		openBaoClient, openBaoErr := c.OpenBaoClient()
		if openBaoErr != nil {
			err = openBaoErr

			return
		}

		c.healthService.instance, err = healthcheck.New(
			c.Config(),
			db,
			openBaoClient,
			http.DefaultClient,
		)
	})

	if err != nil {
		c.healthService.once = sync.Once{}

		return nil, fmt.Errorf("initializing the health service: %w", err)
	}

	return c.healthService.instance, nil
}
