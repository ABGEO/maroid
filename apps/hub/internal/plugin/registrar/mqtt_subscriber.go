package registrar

import (
	"fmt"
	"strings"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// MQTTSubscriberRegistrar is responsible for registering plugin MQTT subscribers.
type MQTTSubscriberRegistrar struct {
	registry *registry.MQTTSubscriberRegistry
}

var _ Registrar = (*MQTTSubscriberRegistrar)(nil)

// NewMQTTSubscriberRegistrar creates a new MQTTSubscriberRegistrar.
func NewMQTTSubscriberRegistrar(reg *registry.MQTTSubscriberRegistry) *MQTTSubscriberRegistrar {
	return &MQTTSubscriberRegistrar{registry: reg}
}

// Name returns the name of the registrar.
func (r *MQTTSubscriberRegistrar) Name() string {
	return "mqtt_subscriber"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *MQTTSubscriberRegistrar) Supports(plugin pluginapi.Plugin) bool {
	_, ok := plugin.(pluginapi.MQTTSubscriberPlugin)

	return ok
}

// Register handles the registration of a plugin's MQTT subscribers.
func (r *MQTTSubscriberRegistrar) Register(plugin pluginapi.Plugin) error {
	id := plugin.Meta().ID

	mqttPlugin, ok := plugin.(pluginapi.MQTTSubscriberPlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support MQTTSubscriber capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	namespace := strings.ReplaceAll(id.String(), ".", "/")

	subscribers, err := mqttPlugin.MQTTSubscribers()
	if err != nil {
		return fmt.Errorf("retrieving MQTT subscribers for plugin %s: %w", id, err)
	}

	for _, sub := range subscribers {
		meta := sub.Meta()

		if err = validateMQTTTopic(meta.Topic); err != nil {
			return fmt.Errorf("invalid topic for subscriber %s in plugin %s: %w", meta.ID, id, err)
		}

		effectiveTopic := namespace + "/" + meta.Topic

		if err = r.registry.Register(effectiveTopic, sub); err != nil {
			return fmt.Errorf(
				"registering MQTT subscriber %s for plugin %s: %w",
				meta.ID,
				id,
				err,
			)
		}
	}

	return nil
}

func validateMQTTTopic(topic string) error {
	if topic == "" {
		return fmt.Errorf("%w: topic must not be empty", errs.ErrInvalidMQTTTopic)
	}

	if strings.HasPrefix(topic, "/") {
		return fmt.Errorf("%w: topic must not start with /", errs.ErrInvalidMQTTTopic)
	}

	if strings.HasPrefix(topic, "$") {
		return fmt.Errorf(
			"%w: topic must not start with $ (reserved for broker internals)",
			errs.ErrInvalidMQTTTopic,
		)
	}

	return nil
}
