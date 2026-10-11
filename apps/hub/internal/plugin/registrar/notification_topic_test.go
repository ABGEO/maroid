package registrar_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/registrar"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// notifyingProbe is a plugin that declares the notification topics that the test gives it.
type notifyingProbe struct {
	topics []pluginapi.NotificationTopic
}

var _ pluginapi.NotifyingPlugin = (*notifyingProbe)(nil)

func (p *notifyingProbe) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID("dev.maroid.probe"),
		Version:    pluginVersion,
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *notifyingProbe) NotificationTopics() ([]pluginapi.NotificationTopic, error) {
	return p.topics, nil
}

const (
	reminderID   = "reminder"
	reminderName = "Reminder"
)

func newTopicRegistrar() (
	*registrar.NotificationTopicRegistrar,
	*registry.NotificationTopicRegistry,
	*registry.CapabilityRegistry,
) {
	topics := registry.NewNotificationTopicRegistry()
	capabilities := registry.NewCapabilityRegistry()

	return registrar.NewNotificationTopicRegistrar(topics, capabilities), topics, capabilities
}

// NOTIF-SC-001: The registrar stores each declared topic and records the capability.
func TestNotificationTopicRegistrarStoresTheTopics(t *testing.T) {
	t.Parallel()

	reg, topics, capabilities := newTopicRegistrar()
	plugin := &notifyingProbe{topics: []pluginapi.NotificationTopic{
		{ID: reminderID, Name: reminderName},
	}}

	require.NoError(t, reg.Register(plugin))

	entries := topics.Of("dev.maroid.probe")
	require.Len(t, entries, 1)
	require.Equal(t, reminderID, entries[0].Topic.ID)
	require.Equal(t, reminderName, entries[0].Topic.Name)

	entry, found := topics.Get("dev.maroid.probe", reminderID)
	require.True(t, found)
	require.Equal(t, "dev.maroid.probe", entry.PluginID.String())

	require.Equal(
		t,
		[]string{reminderID},
		capabilities.Of("dev.maroid.probe")[registry.CapNotificationTopics],
	)
}

// NOTIF-SC-002: A plugin that declares a topic that breaks a rule fails to load,
// and the registry holds no topic of it.
func TestNotificationTopicRegistrarRefusesAnInvalidDeclaration(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		topics []pluginapi.NotificationTopic
		want   error
	}{
		"repeated id": {
			topics: []pluginapi.NotificationTopic{
				{ID: reminderID, Name: reminderName},
				{ID: reminderID, Name: "Second"},
			},
			want: errs.ErrNotificationTopicAlreadyRegistered,
		},
		"bad identifier": {
			topics: []pluginapi.NotificationTopic{{ID: "Reminder-1", Name: "Reminder"}},
			want:   errs.ErrInvalidNotificationTopic,
		},
		"long name": {
			topics: []pluginapi.NotificationTopic{{ID: reminderID, Name: strings.Repeat("a", 65)}},
			want:   errs.ErrInvalidNotificationTopic,
		},
		"empty name": {
			topics: []pluginapi.NotificationTopic{{ID: reminderID, Name: "   "}},
			want:   errs.ErrInvalidNotificationTopic,
		},
		"long description": {
			topics: []pluginapi.NotificationTopic{
				{ID: reminderID, Name: "Reminder", Description: strings.Repeat("a", 281)},
			},
			want: errs.ErrInvalidNotificationTopic,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reg, topics, _ := newTopicRegistrar()

			err := reg.Register(&notifyingProbe{topics: tc.topics})

			require.ErrorIs(t, err, tc.want)
			require.ErrorContains(t, err, "dev.maroid.probe")
			require.Empty(t, topics.Of("dev.maroid.probe"))
		})
	}
}
