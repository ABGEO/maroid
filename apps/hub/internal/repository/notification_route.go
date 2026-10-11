package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/model"
)

const notificationRouteColumns = `id, workspace_id, provider_id, plugin_id, topic_id, ` +
	`destination, enabled, created_at, updated_at`

// NotificationRouteRepository defines the data access contract for the notification
// routes of the acting workspace.
type NotificationRouteRepository interface {
	Create(ctx context.Context, route *model.NotificationRoute) (*model.NotificationRoute, error)
	ListDeliverable(
		ctx context.Context,
		pluginID string,
		topicID string,
	) ([]model.DeliverableRoute, error)
}

// NotificationRoute is a SQL based implementation of NotificationRouteRepository.
type NotificationRoute struct {
	tx *sqlx.Tx
}

var _ NotificationRouteRepository = (*NotificationRoute)(nil)

// NewNotificationRoute creates a new NotificationRoute repository instance.
func NewNotificationRoute(tx *sqlx.Tx) *NotificationRoute {
	return &NotificationRoute{tx: tx}
}

// Create stores a notification route in the acting workspace, and answers the stored row.
func (r *NotificationRoute) Create(
	ctx context.Context,
	route *model.NotificationRoute,
) (*model.NotificationRoute, error) {
	var stored model.NotificationRoute

	query := `
		INSERT INTO public.notification_routes (provider_id, plugin_id, topic_id, destination, enabled)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + notificationRouteColumns + `;`

	err := r.tx.GetContext(
		ctx, &stored, query,
		route.ProviderID, route.PluginID, route.TopicID, route.Destination, route.Enabled,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting NotificationRoute: %w", err)
	}

	return &stored, nil
}

// ListDeliverable answers each enabled route of the topic whose provider is enabled,
// in the order of the route identifiers.
func (r *NotificationRoute) ListDeliverable(
	ctx context.Context,
	pluginID string,
	topicID string,
) ([]model.DeliverableRoute, error) {
	routes := []model.DeliverableRoute{}

	query := `
		SELECT r.id AS route_id, r.provider_id, p.kind, p.fields AS provider_fields, r.destination
		FROM public.notification_routes r
		JOIN public.notification_providers p
		  ON p.workspace_id = r.workspace_id AND p.id = r.provider_id
		WHERE r.plugin_id = $1 AND r.topic_id = $2 AND r.enabled AND p.enabled
		ORDER BY r.id;`

	if err := r.tx.SelectContext(ctx, &routes, query, pluginID, topicID); err != nil {
		return nil, fmt.Errorf("listing the deliverable NotificationRoutes: %w", err)
	}

	return routes, nil
}
