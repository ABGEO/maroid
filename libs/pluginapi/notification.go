package pluginapi

import (
	"context"
	"errors"
	"fmt"
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
	Plugin
	NotificationTopics() ([]NotificationTopic, error)
}

// NotificationSender delivers a notification of one plugin through the notification
// routes of the acting workspace. The hub implements it, and Host hands it to a plugin.
type NotificationSender interface {
	Send(ctx context.Context, pluginID *PluginID, topicID string, msg Message) error
}

// PluginNotifier binds a NotificationSender to one plugin.
type PluginNotifier struct {
	sender   NotificationSender
	pluginID *PluginID
}

// NewPluginNotifier creates a new PluginNotifier instance.
func NewPluginNotifier(sender NotificationSender, pluginID *PluginID) *PluginNotifier {
	return &PluginNotifier{
		sender:   sender,
		pluginID: pluginID,
	}
}

// Notify sends a notification of the topic in the acting workspace of the context.
// It returns nil when the workspace routes the topic nowhere.
func (n *PluginNotifier) Notify(ctx context.Context, topicID string, msg Message) error {
	if err := n.sender.Send(ctx, n.pluginID, topicID, msg); err != nil {
		return fmt.Errorf("sending the notification %q: %w", topicID, err)
	}

	return nil
}
