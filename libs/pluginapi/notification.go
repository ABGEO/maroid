package pluginapi

import (
	"context"
	"errors"
)

// ErrNotificationTopicUndeclared reports a send of a notification topic that the plugin
// does not declare.
var ErrNotificationTopicUndeclared = errors.New(
	"notification: the plugin does not declare the topic",
)

// Attachment is a file that a notification carries.
type Attachment struct {
	Filename string
	Content  []byte
	MIMEType string
}

// Message is the content of one notification.
type Message struct {
	Title       string
	Body        string
	Attachments []Attachment
}

// NotificationTopic is one kind of notification that a plugin sends. A manager routes
// it in each workspace.
type NotificationTopic struct {
	ID          string
	Name        string
	Description string
}

// NotifyingPlugin is a plugin that declares the notification topics that it sends.
type NotifyingPlugin interface {
	NotificationTopics() ([]NotificationTopic, error)
}

// PluginNotifier sends the notifications of one plugin.
type PluginNotifier interface {
	// Notify sends a notification of the topic in the acting workspace of the context.
	// It returns nil when the workspace routes the topic nowhere, and
	// ErrNotificationTopicUndeclared when the plugin does not declare the topic.
	Notify(ctx context.Context, topicID string, msg Message) error
}
