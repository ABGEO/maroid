package notification

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// TopicSource gives the notification topics that the plugins declare.
// registry.NotificationTopicRegistry satisfies it.
type TopicSource interface {
	Get(pluginID string, topicID string) (registry.NotificationTopicEntry, bool)
}

// Dispatcher delivers a notification of a plugin in the acting workspace.
type Dispatcher struct {
	logger *slog.Logger
	topics TopicSource
}

var _ pluginapi.NotificationSender = (*Dispatcher)(nil)

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(logger *slog.Logger, topics TopicSource) *Dispatcher {
	return &Dispatcher{
		logger: logger.With(slog.String("component", "notification")),
		topics: topics,
	}
}

// Send delivers the notification through each enabled notification route of the topic.
// A topic that the plugin does not declare is an error. A unit of work with no acting
// workspace and an unrouted topic each drop the notification and return nil.
func (d *Dispatcher) Send(
	ctx context.Context,
	pluginID *pluginapi.PluginID,
	topicID string,
	_ pluginapi.Message,
) error {
	if _, declared := d.topics.Get(pluginID.String(), topicID); !declared {
		return fmt.Errorf("%w: %s %s", pluginapi.ErrNotificationTopicUndeclared, pluginID, topicID)
	}

	logger := d.logger.With(
		slog.String("plugin", pluginID.String()),
		slog.String("topic", topicID),
	)

	workspaceID := pluginapi.ActingWorkspaceFromContext(ctx)
	if workspaceID == "" {
		logger.WarnContext(ctx, "notification has no workspace")

		return nil
	}

	logger.InfoContext(
		ctx,
		"notification unrouted",
		slog.String("workspace_id", workspaceID),
	)

	return nil
}
