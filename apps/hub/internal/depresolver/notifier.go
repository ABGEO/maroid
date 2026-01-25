package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/libs/notifier/dispatcher"
	"github.com/abgeo/maroid/libs/notifier/registry"
	"github.com/abgeo/maroid/libs/notifier/transport/telegram"
)

// NotifierRegistry initializes and returns the notifier registry instance.
func (c *Container) NotifierRegistry() (*registry.SchemeRegistry, error) {
	c.notifierRegistry.mu.Lock()
	defer c.notifierRegistry.mu.Unlock()

	var err error

	c.notifierRegistry.once.Do(func() {
		c.notifierRegistry.instance = registry.New()
		err = registerNotifiers(c.notifierRegistry.instance)
	})

	if err != nil {
		c.notifierRegistry.once = sync.Once{}

		return nil, fmt.Errorf("failed to initialize notifier registry: %w", err)
	}

	return c.notifierRegistry.instance, nil
}

// NotifierDispatcher initializes and returns the notifier dispatcher instance.
func (c *Container) NotifierDispatcher() (*dispatcher.ChannelDispatcher, error) {
	c.notifierDispatcher.mu.Lock()
	defer c.notifierDispatcher.mu.Unlock()

	var err error

	c.notifierDispatcher.once.Do(func() {
		var reg registry.Registry

		reg, err = c.NotifierRegistry()
		if err != nil {
			return
		}

		c.notifierDispatcher.instance, err = dispatcher.NewDispatcher(
			&c.Config().Notifier,
			c.Logger(),
			reg,
		)
	})

	if err != nil {
		c.notifierDispatcher.once = sync.Once{}

		return nil, fmt.Errorf("failed to initialize notifier dispatcher: %w", err)
	}

	return c.notifierDispatcher.instance, nil
}

func registerNotifiers(reg registry.Registry) error {
	registrations := []func(registry.Registry) error{
		telegram.Register,
	}

	for i, register := range registrations {
		if err := register(reg); err != nil {
			return fmt.Errorf("registration %d failed: %w", i, err)
		}
	}

	return nil
}
