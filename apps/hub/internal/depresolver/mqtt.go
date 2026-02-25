package depresolver

import (
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// MQTTSubscriberRegistry initializes and returns the MQTT subscriber registry instance.
func (c *Container) MQTTSubscriberRegistry() (*registry.MQTTSubscriberRegistry, error) {
	c.mqttSubscriberRegistry.mu.Lock()
	defer c.mqttSubscriberRegistry.mu.Unlock()

	c.mqttSubscriberRegistry.once.Do(func() {
		c.mqttSubscriberRegistry.instance = registry.NewMQTTSubscriberRegistry()
	})

	return c.mqttSubscriberRegistry.instance, nil
}
