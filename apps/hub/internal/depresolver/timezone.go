package depresolver

import "github.com/abgeo/maroid/apps/hub/internal/timezone"

// TimeZones initializes and returns the one cache of the locations of the time zones.
func (c *Container) TimeZones() *timezone.Cache {
	c.timeZones.once.Do(func() {
		c.timeZones.instance = timezone.New(c.logger)
	})

	return c.timeZones.instance
}
