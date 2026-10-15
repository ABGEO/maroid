package host

import (
	"context"
	"fmt"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// NotificationSender delivers a notification of any plugin through the notification
// routes of the acting workspace.
type NotificationSender interface {
	Send(
		ctx context.Context,
		pluginID *pluginapi.PluginID,
		topicID string,
		msg pluginapi.Message,
	) error
}

type boundNotifier struct {
	sender   NotificationSender
	pluginID *pluginapi.PluginID
}

var _ pluginapi.PluginNotifier = (*boundNotifier)(nil)

func (n *boundNotifier) Notify(ctx context.Context, topicID string, msg pluginapi.Message) error {
	if err := n.sender.Send(ctx, n.pluginID, topicID, msg); err != nil {
		return fmt.Errorf("sending the notification %q: %w", topicID, err)
	}

	return nil
}
