package depresolver

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/robfig/cron/v3"

	"github.com/abgeo/maroid/apps/hub/internal/idempotency"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// Cron initializes and returns the cron scheduler instance.
func (c *Container) Cron() *cron.Cron {
	c.cron.once.Do(func() {
		parser := cron.NewParser(
			cron.SecondOptional |
				cron.Minute |
				cron.Hour |
				cron.Dom |
				cron.Month |
				cron.Dow,
		)
		c.cron.instance = cron.New(cron.WithParser(parser))
	})

	return c.cron.instance
}

// CronRegistry initializes and returns the cron registry instance. It holds the
// jobs of the hub before a plugin adds one, the way MigrationRegistry holds the
// core migrations.
func (c *Container) CronRegistry() (*registry.CronRegistry, error) {
	c.cronRegistry.mu.Lock()
	defer c.cronRegistry.mu.Unlock()

	var err error

	c.cronRegistry.once.Do(func() {
		c.cronRegistry.instance = registry.NewCronRegistry()

		var dbInstance *sqlx.DB

		dbInstance, err = c.Database()
		if err != nil {
			return
		}

		err = c.cronRegistry.instance.Register(coreCronJobs(c.Logger(), dbInstance)...)
	})

	if err != nil {
		c.cronRegistry.once = sync.Once{}

		return nil, fmt.Errorf("initializing the cron registry: %w", err)
	}

	return c.cronRegistry.instance, nil
}

// coreCronJobs returns the jobs that the hub runs for itself. A plugin adds its
// own through the registrar, and this list is the one the hub owns.
func coreCronJobs(logger *slog.Logger, db *sqlx.DB) []pluginapi.CronJob {
	return []pluginapi.CronJob{
		idempotency.NewSweep(logger, db),
	}
}
