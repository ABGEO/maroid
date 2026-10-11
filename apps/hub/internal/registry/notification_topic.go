package registry

import (
	"fmt"
	"slices"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// NotificationTopicEntry is one notification topic of one plugin.
type NotificationTopicEntry struct {
	PluginID *pluginapi.PluginID
	Topic    pluginapi.NotificationTopic
}

// NotificationTopicRegistry holds the notification topics of every plugin, in the
// order that each plugin declares them.
type NotificationTopicRegistry struct {
	entries map[string][]NotificationTopicEntry
}

// NewNotificationTopicRegistry creates a new NotificationTopicRegistry.
func NewNotificationTopicRegistry() *NotificationTopicRegistry {
	return &NotificationTopicRegistry{entries: make(map[string][]NotificationTopicEntry)}
}

// Register stores the topics of one plugin, all of them or none. An identifier that
// the plugin already holds, or that the topics hold twice, refuses the whole call.
func (r *NotificationTopicRegistry) Register(
	pluginID *pluginapi.PluginID,
	topics ...pluginapi.NotificationTopic,
) error {
	id := pluginID.String()
	held := r.entries[id]
	seen := make(map[string]struct{}, len(held)+len(topics))

	for _, entry := range held {
		seen[entry.Topic.ID] = struct{}{}
	}

	added := make([]NotificationTopicEntry, 0, len(topics))

	for _, topic := range topics {
		if _, repeated := seen[topic.ID]; repeated {
			return fmt.Errorf("%w: %s %s", errs.ErrNotificationTopicAlreadyRegistered, id, topic.ID)
		}

		seen[topic.ID] = struct{}{}
		added = append(added, NotificationTopicEntry{PluginID: pluginID, Topic: topic})
	}

	r.entries[id] = append(held, added...)

	return nil
}

// Get returns one topic of one plugin.
func (r *NotificationTopicRegistry) Get(
	pluginID string,
	topicID string,
) (NotificationTopicEntry, bool) {
	index := slices.IndexFunc(r.entries[pluginID], func(entry NotificationTopicEntry) bool {
		return entry.Topic.ID == topicID
	})
	if index < 0 {
		return NotificationTopicEntry{}, false
	}

	return r.entries[pluginID][index], true
}

// Of returns the topics of one plugin, in the declared order.
func (r *NotificationTopicRegistry) Of(pluginID string) []NotificationTopicEntry {
	return slices.Clone(r.entries[pluginID])
}
