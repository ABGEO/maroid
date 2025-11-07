package depresolver

import (
	"github.com/robfig/cron/v3"
)

// Cron returns the cron scheduler instance.
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
