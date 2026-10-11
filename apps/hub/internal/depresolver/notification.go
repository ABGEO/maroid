package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/notification"
	"github.com/abgeo/maroid/apps/hub/internal/notification/telegram"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// NotificationTopicRegistry initializes and returns the notification topic registry instance.
func (c *Container) NotificationTopicRegistry() *registry.NotificationTopicRegistry {
	c.notificationTopicRegistry.once.Do(func() {
		c.notificationTopicRegistry.instance = registry.NewNotificationTopicRegistry()
	})

	return c.notificationTopicRegistry.instance
}

// NotificationKinds initializes and returns the registry of the notification provider kinds.
func (c *Container) NotificationKinds() (*notification.KindRegistry, error) {
	c.notificationKinds.mu.Lock()
	defer c.notificationKinds.mu.Unlock()

	var err error

	c.notificationKinds.once.Do(func() {
		kinds := notification.NewKindRegistry()
		if err = kinds.Register(telegram.New()); err != nil {
			return
		}

		c.notificationKinds.instance = kinds
	})

	if err != nil {
		c.notificationKinds.once = sync.Once{}

		return nil, fmt.Errorf("initializing the notification kinds: %w", err)
	}

	return c.notificationKinds.instance, nil
}

// NotificationDispatcher initializes and returns the notification dispatcher instance.
func (c *Container) NotificationDispatcher() (*notification.Dispatcher, error) {
	c.notificationDispatcher.mu.Lock()
	defer c.notificationDispatcher.mu.Unlock()

	var err error

	c.notificationDispatcher.once.Do(func() {
		c.notificationDispatcher.instance, err = c.buildNotificationDispatcher()
	})

	if err != nil {
		c.notificationDispatcher.once = sync.Once{}

		return nil, fmt.Errorf("initializing the notification dispatcher: %w", err)
	}

	return c.notificationDispatcher.instance, nil
}

func (c *Container) buildNotificationDispatcher() (*notification.Dispatcher, error) {
	db, err := c.Database()
	if err != nil {
		return nil, err
	}

	kinds, err := c.NotificationKinds()
	if err != nil {
		return nil, err
	}

	cipher, err := c.SecretCipher()
	if err != nil {
		return nil, err
	}

	return notification.NewDispatcher(
		c.Logger(),
		c.NotificationTopicRegistry(),
		db,
		kinds,
		cipher,
	), nil
}
