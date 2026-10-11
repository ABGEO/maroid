package registrar

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	maxTopicNameLength        = 64
	maxTopicDescriptionLength = 280
)

// topicIDPattern keeps a topic identifier safe in a path and in a log.
var topicIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// NotificationTopicRegistrar registers the notification topics that a plugin declares.
type NotificationTopicRegistrar struct {
	registry     *registry.NotificationTopicRegistry
	capabilities *registry.CapabilityRegistry
}

var _ Registrar = (*NotificationTopicRegistrar)(nil)

// NewNotificationTopicRegistrar creates a new NotificationTopicRegistrar.
func NewNotificationTopicRegistrar(
	reg *registry.NotificationTopicRegistry,
	capabilities *registry.CapabilityRegistry,
) *NotificationTopicRegistrar {
	return &NotificationTopicRegistrar{registry: reg, capabilities: capabilities}
}

// Name returns the name of the registrar.
func (r *NotificationTopicRegistrar) Name() string {
	return "notification_topic"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *NotificationTopicRegistrar) Supports(plugin pluginapi.Plugin) bool {
	_, ok := plugin.(pluginapi.NotifyingPlugin)

	return ok
}

// Register stores every notification topic of the plugin, after it trims each name.
func (r *NotificationTopicRegistrar) Register(plugin pluginapi.Plugin) error {
	id := plugin.Meta().ID

	notifying, ok := plugin.(pluginapi.NotifyingPlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support NotificationTopic capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	declared, err := notifying.NotificationTopics()
	if err != nil {
		return fmt.Errorf("retrieving the notification topics for plugin %s: %w", id, err)
	}

	topics := make([]pluginapi.NotificationTopic, 0, len(declared))
	identifiers := make([]string, 0, len(declared))

	for _, topic := range declared {
		topic.Name = strings.TrimSpace(topic.Name)

		if err = validateTopic(topic); err != nil {
			return fmt.Errorf("plugin %s: %w", id, err)
		}

		topics = append(topics, topic)
		identifiers = append(identifiers, topic.ID)
	}

	if err = r.registry.Register(id, topics...); err != nil {
		return fmt.Errorf("registering the notification topics for plugin %s: %w", id, err)
	}

	r.capabilities.Record(id, registry.CapNotificationTopics, identifiers)

	return nil
}

func validateTopic(topic pluginapi.NotificationTopic) error {
	switch nameLength := utf8.RuneCountInString(topic.Name); {
	case !topicIDPattern.MatchString(topic.ID):
		return fmt.Errorf("%w: the identifier %q", errs.ErrInvalidNotificationTopic, topic.ID)
	case nameLength == 0 || nameLength > maxTopicNameLength:
		return fmt.Errorf("%w: the name of %q", errs.ErrInvalidNotificationTopic, topic.ID)
	case utf8.RuneCountInString(topic.Description) > maxTopicDescriptionLength:
		return fmt.Errorf("%w: the description of %q", errs.ErrInvalidNotificationTopic, topic.ID)
	default:
		return nil
	}
}
