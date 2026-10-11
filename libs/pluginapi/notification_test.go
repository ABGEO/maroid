package pluginapi_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// senderRecorder is a NotificationSender that keeps the arguments of the last call.
type senderRecorder struct {
	pluginID *pluginapi.PluginID
	topicID  string
	msg      pluginapi.Message
	err      error
}

func (s *senderRecorder) Send(
	_ context.Context,
	pluginID *pluginapi.PluginID,
	topicID string,
	msg pluginapi.Message,
) error {
	s.pluginID = pluginID
	s.topicID = topicID
	s.msg = msg

	return s.err
}

// NOTIF-SC-003: A plugin names a notification topic, and the bind names the plugin.
// This part covers the bind. The dispatcher test covers the undeclared topic.
func TestNotifyNamesThePluginThatHoldsTheNotifier(t *testing.T) {
	t.Parallel()

	sender := &senderRecorder{}
	notifier := pluginapi.NewPluginNotifier(sender, pluginapi.ParsePluginID("dev.maroid.jasmine"))
	msg := pluginapi.Message{Title: "Plant care due", Body: "E001 Office"}

	err := notifier.Notify(t.Context(), "care_reminder", msg)

	require.NoError(t, err)
	require.Equal(t, "dev.maroid.jasmine", sender.pluginID.String())
	require.Equal(t, "care_reminder", sender.topicID)
	require.Equal(t, msg, sender.msg)
}

// NOTIF-SC-003: An undeclared topic reaches the plugin as ErrNotificationTopicUndeclared.
func TestNotifyReportsAnUndeclaredTopic(t *testing.T) {
	t.Parallel()

	sender := &senderRecorder{err: pluginapi.ErrNotificationTopicUndeclared}
	notifier := pluginapi.NewPluginNotifier(sender, pluginapi.ParsePluginID("dev.maroid.jasmine"))

	err := notifier.Notify(t.Context(), "invoice", pluginapi.Message{})

	require.ErrorIs(t, err, pluginapi.ErrNotificationTopicUndeclared)
}
