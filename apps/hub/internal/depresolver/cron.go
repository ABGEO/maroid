package depresolver

import (
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
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

// CronRegistry initializes and returns the cron registry instance.
func (c *Container) CronRegistry() (*registry.CronRegistry, error) {
	c.cronRegistry.mu.Lock()
	defer c.cronRegistry.mu.Unlock()

	var err error

	c.cronRegistry.once.Do(func() {
		c.cronRegistry.instance = registry.NewCronRegistry()
	})

	if err != nil {
		c.cronRegistry.once = sync.Once{}

		return nil, fmt.Errorf("failed to initialize cron registry: %w", err)
	}

	return c.cronRegistry.instance, nil
}
