package host_test

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// senderRecorder is a notification sender that keeps the arguments of the last call.
// It refuses each topic outside declared.
type senderRecorder struct {
	declared []string
	pluginID *pluginapi.PluginID
	topicID  string
	msg      pluginapi.Message
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

	if slices.Contains(s.declared, topicID) {
		return nil
	}

	return pluginapi.ErrNotificationTopicUndeclared
}

func notifierOf(t *testing.T, sender *senderRecorder, pluginID string) pluginapi.PluginNotifier {
	t.Helper()

	notifier, err := bind(t, host.Dependencies{Notifier: sender}, pluginID).Notifier()
	require.NoError(t, err)

	return notifier
}

// NOTIF-SC-003: A plugin names a notification topic, and the bind names the plugin.
// This part covers the bind. The dispatcher test covers the undeclared topic.
func TestNotifyNamesThePluginThatHoldsTheNotifier(t *testing.T) {
	t.Parallel()

	sender := &senderRecorder{declared: []string{"care_reminder"}}
	msg := pluginapi.Message{Title: "Plant care due", Body: "E001 Office"}

	err := notifierOf(t, sender, "dev.maroid.jasmine").Notify(t.Context(), "care_reminder", msg)

	require.NoError(t, err)
	require.Equal(t, "dev.maroid.jasmine", sender.pluginID.String())
	require.Equal(t, "care_reminder", sender.topicID)
	require.Equal(t, msg, sender.msg)
}

// NOTIF-SC-003: An undeclared topic reaches the plugin as ErrNotificationTopicUndeclared.
// PLGBIND-SC-013: A topic of another plugin is undeclared for the bound plugin.
func TestNotifyReportsAnUndeclaredTopic(t *testing.T) {
	t.Parallel()

	sender := &senderRecorder{declared: []string{"new-reading"}}
	notifier := notifierOf(t, sender, "dev.maroid.gwp")

	require.NoError(t, notifier.Notify(t.Context(), "new-reading", pluginapi.Message{}))
	require.Equal(t, "dev.maroid.gwp", sender.pluginID.String())

	err := notifier.Notify(t.Context(), "new-bill", pluginapi.Message{})

	require.ErrorIs(t, err, pluginapi.ErrNotificationTopicUndeclared)
	require.Equal(t, "dev.maroid.gwp", sender.pluginID.String())
}
