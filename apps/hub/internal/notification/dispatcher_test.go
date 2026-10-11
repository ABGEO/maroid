package notification_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/notification"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	probeID       = "dev.maroid.probe"
	reminderTopic = "reminder"
	reportTopic   = "report"
)

func newDispatcher(t *testing.T) (*notification.Dispatcher, *bytes.Buffer) {
	t.Helper()

	topics := registry.NewNotificationTopicRegistry()
	require.NoError(t, topics.Register(
		pluginapi.ParsePluginID(probeID),
		pluginapi.NotificationTopic{ID: reminderTopic, Name: "Reminder"},
		pluginapi.NotificationTopic{ID: reportTopic, Name: "Report"},
	))

	lines := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(lines, &slog.HandlerOptions{Level: slog.LevelDebug}))

	return notification.NewDispatcher(
		logger,
		topics,
		nil,
		notification.NewKindRegistry(),
		nil,
	), lines
}

func logLines(t *testing.T, lines *bytes.Buffer) []map[string]any {
	t.Helper()

	var records []map[string]any

	for line := range bytes.SplitSeq(bytes.TrimSpace(lines.Bytes()), []byte("\n")) {
		if len(line) == 0 {
			continue
		}

		record := map[string]any{}
		require.NoError(t, json.Unmarshal(line, &record))
		records = append(records, record)
	}

	return records
}

// NOTIF-SC-003: A topic that the plugin does not declare is a fault of the plugin.
func TestSendRefusesAnUndeclaredTopic(t *testing.T) {
	t.Parallel()

	dispatcher, _ := newDispatcher(t)
	ctx := pluginapi.ContextWithActingWorkspace(t.Context(), "0199d4a0-0000-7000-8000-000000000001")

	err := dispatcher.Send(ctx, pluginapi.ParsePluginID(probeID), "invoice", pluginapi.Message{})

	require.ErrorIs(t, err, pluginapi.ErrNotificationTopicUndeclared)
}

// NOTIF-SC-022: A send with no acting workspace delivers nothing, logs a warning,
// and reports no failure.
func TestSendWithNoWorkspaceDropsTheNotification(t *testing.T) {
	t.Parallel()

	dispatcher, lines := newDispatcher(t)
	ctx := pluginapi.ContextWithActingUser(t.Context(), "0199d4a0-0000-7000-8000-00000000000a")

	err := dispatcher.Send(
		ctx,
		pluginapi.ParsePluginID(probeID),
		reminderTopic,
		pluginapi.Message{},
	)

	require.NoError(t, err)

	records := logLines(t, lines)
	require.Len(t, records, 1)
	require.Equal(t, "WARN", records[0]["level"])
	require.Equal(t, "notification has no workspace", records[0]["msg"])
	require.Equal(t, probeID, records[0]["plugin"])
	require.Equal(t, reminderTopic, records[0]["topic"])
}
