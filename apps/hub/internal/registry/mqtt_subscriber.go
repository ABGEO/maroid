package registry

import (
	"fmt"
	"maps"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// MQTTSubscriberRegistry is a registry for MQTT subscribers.
// Subscribers are keyed by their effective topic (namespace/relative-topic).
type MQTTSubscriberRegistry struct {
	subscribers map[string]pluginapi.MQTTSubscriber
}

// NewMQTTSubscriberRegistry creates a new MQTTSubscriberRegistry.
func NewMQTTSubscriberRegistry() *MQTTSubscriberRegistry {
	return &MQTTSubscriberRegistry{
		subscribers: make(map[string]pluginapi.MQTTSubscriber),
	}
}

// Register stores a subscriber under its effective topic.
// Returns ErrMQTTSubscriberAlreadyRegistered if the topic is already taken.
func (r *MQTTSubscriberRegistry) Register(
	effectiveTopic string,
	sub pluginapi.MQTTSubscriber,
) error {
	if _, exists := r.subscribers[effectiveTopic]; exists {
		return fmt.Errorf("%w: %s", errs.ErrMQTTSubscriberAlreadyRegistered, effectiveTopic)
	}

	r.subscribers[effectiveTopic] = sub

	return nil
}

// All returns a copy of the effective topic to subscriber map.
func (r *MQTTSubscriberRegistry) All() map[string]pluginapi.MQTTSubscriber {
	out := make(map[string]pluginapi.MQTTSubscriber, len(r.subscribers))

	maps.Copy(out, r.subscribers)

	return out
}
