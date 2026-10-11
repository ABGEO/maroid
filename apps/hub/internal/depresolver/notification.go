package depresolver

import (
	"github.com/abgeo/maroid/apps/hub/internal/notification"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// NotificationTopicRegistry initializes and returns the notification topic registry instance.
func (c *Container) NotificationTopicRegistry() *registry.NotificationTopicRegistry {
	c.notificationTopicRegistry.once.Do(func() {
		c.notificationTopicRegistry.instance = registry.NewNotificationTopicRegistry()
	})

	return c.notificationTopicRegistry.instance
}

// NotificationDispatcher initializes and returns the notification dispatcher instance.
func (c *Container) NotificationDispatcher() (*notification.Dispatcher, error) {
	c.notificationDispatcher.mu.Lock()
	defer c.notificationDispatcher.mu.Unlock()

	c.notificationDispatcher.once.Do(func() {
		c.notificationDispatcher.instance = notification.NewDispatcher(
			c.Logger(),
			c.NotificationTopicRegistry(),
		)
	})

	return c.notificationDispatcher.instance, nil
}
