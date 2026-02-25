package pluginapi

import "context"

// MQTTSubscriberMeta holds metadata for an MQTT subscriber.
type MQTTSubscriberMeta struct {
	ID    string // unique identifier for the subscriber
	Topic string // relative topic pattern; wildcards + and # are allowed
	QoS   byte   // 0, 1, or 2
}

// MQTTSubscriberPlugin is a plugin that can register MQTT topic subscribers.
type MQTTSubscriberPlugin interface {
	Plugin
	MQTTSubscribers() ([]MQTTSubscriber, error)
}

// MQTTSubscriber handles messages arriving on a declared topic pattern.
type MQTTSubscriber interface {
	Meta() MQTTSubscriberMeta
	Handle(ctx context.Context, topic string, payload []byte) error
}
