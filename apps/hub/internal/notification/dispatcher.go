package notification

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// TopicSource gives the notification topics that the plugins declare.
// registry.NotificationTopicRegistry satisfies it.
type TopicSource interface {
	Get(pluginID string, topicID string) (registry.NotificationTopicEntry, bool)
}

// Dispatcher delivers a notification of a plugin in the acting workspace. It holds
// no cache, so each send reads the routes and the secrets that the last save stored.
type Dispatcher struct {
	logger *slog.Logger
	topics TopicSource
	db     *sqlx.DB
	kinds  *KindRegistry
	cipher secret.Cipher
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(
	logger *slog.Logger,
	topics TopicSource,
	db *sqlx.DB,
	kinds *KindRegistry,
	cipher secret.Cipher,
) *Dispatcher {
	return &Dispatcher{
		logger: logger.With(slog.String("component", "notification")),
		topics: topics,
		db:     db,
		kinds:  kinds,
		cipher: cipher,
	}
}

// Send delivers the notification through each enabled notification route of the topic,
// one after the other. It returns an error for a topic that the plugin does not
// declare, or when every route failed. A unit of work with no acting workspace and an
// unrouted topic each drop the notification and return nil.
func (d *Dispatcher) Send(
	ctx context.Context,
	pluginID *pluginapi.PluginID,
	topicID string,
	msg pluginapi.Message,
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

	logger = logger.With(slog.String("workspace_id", workspaceID))

	var routes []model.DeliverableRoute

	err := database.WithScopeTx(ctx, d.db, func(tx *sqlx.Tx) error {
		var listErr error

		routes, listErr = repository.NewNotificationRoute(tx).ListDeliverable(
			ctx, pluginID.String(), topicID,
		)
		if listErr != nil {
			return fmt.Errorf("listing the deliverable routes: %w", listErr)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("reading the notification routes: %w", err)
	}

	if len(routes) == 0 {
		logger.InfoContext(ctx, "notification unrouted")

		return nil
	}

	return d.deliver(ctx, logger, workspaceID, routes, msg)
}

func (d *Dispatcher) deliver(
	ctx context.Context,
	logger *slog.Logger,
	workspaceID string,
	routes []model.DeliverableRoute,
	msg pluginapi.Message,
) error {
	failures := make([]error, 0, len(routes))

	for _, route := range routes {
		err := d.deliverOne(ctx, workspaceID, route, msg)
		if err == nil {
			continue
		}

		logger.ErrorContext(
			ctx,
			"notification route failed",
			slog.String("route_id", route.RouteID),
			slog.String("provider_id", route.ProviderID),
			slog.String("kind", route.Kind),
			slog.Any("error", err),
		)

		failures = append(
			failures,
			fmt.Errorf("delivering through the route %s: %w", route.RouteID, err),
		)
	}

	if len(failures) < len(routes) {
		return nil
	}

	return fmt.Errorf("every notification route failed: %w", errors.Join(failures...))
}

func (d *Dispatcher) deliverOne(
	ctx context.Context,
	workspaceID string,
	route model.DeliverableRoute,
	msg pluginapi.Message,
) error {
	entry, found := d.kinds.Get(route.Kind)
	if !found {
		return fmt.Errorf(
			"%w: the kind %q is not registered",
			errs.ErrInvalidNotificationKind,
			route.Kind,
		)
	}

	provider, err := Reveal(ctx, d.cipher, workspaceID, route.ProviderFields)
	if err != nil {
		return err
	}

	if err = entry.Kind.Send(ctx, provider, route.Destination, msg); err != nil {
		return fmt.Errorf("sending through %s: %w", route.Kind, err)
	}

	return nil
}

// Reveal returns the plaintext of every stored field of a notification provider. A
// secret takes the key of its workspace.
func Reveal(
	ctx context.Context,
	cipher secret.Cipher,
	workspaceID string,
	fields model.Fields,
) (map[string]any, error) {
	values := make(map[string]any, len(fields))

	for key, entry := range fields {
		if entry.Kind != model.FieldKindSecret {
			values[key] = entry.Value

			continue
		}

		ciphertext, ok := entry.Value.(string)
		if !ok {
			return nil, fmt.Errorf(
				"the field %q holds no protected value: %w", key, errs.ErrProtectionUnavailable,
			)
		}

		plaintext, err := cipher.Decrypt(ctx, secret.WorkspaceKey(workspaceID), ciphertext)
		if err != nil {
			return nil, fmt.Errorf("reading the protected field %q: %w", key, err)
		}

		values[key] = plaintext
	}

	return values, nil
}
