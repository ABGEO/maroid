package depresolver

import (
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
)

// SettingsRegistry initializes and returns the settings schema registry.
func (c *Container) SettingsRegistry() *registry.SettingsRegistry {
	c.settingsRegistry.once.Do(func() {
		c.settingsRegistry.instance = registry.NewSettingsRegistry()
	})

	return c.settingsRegistry.instance
}

// SettingsService initializes and returns the settings service instance.
func (c *Container) SettingsService() (settings.Service, error) {
	c.settingsService.mu.Lock()
	defer c.settingsService.mu.Unlock()

	var err error

	c.settingsService.once.Do(func() {
		var (
			dbInstance *sqlx.DB
			cipher     secret.Cipher
		)

		dbInstance, err = c.Database()
		if err != nil {
			return
		}

		cipher, err = c.SecretCipher()
		if err != nil {
			return
		}

		c.settingsService.instance = settings.NewManager(
			dbInstance,
			c.SettingsRegistry(),
			cipher,
		)
	})

	if err != nil {
		c.settingsService.once = sync.Once{}

		return nil, fmt.Errorf("initializing the settings service: %w", err)
	}

	return c.settingsService.instance, nil
}
