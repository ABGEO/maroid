package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/libs/rest/precondition"
)

const (
	notificationRouteColumns = `id, workspace_id, provider_id, plugin_id, topic_id, ` +
		`destination, enabled, created_at, updated_at`
	notificationRouteKey         = "notification_routes_unique_key"
	notificationRouteProviderKey = "notification_routes_provider_fkey"
)

// NotificationRouteRepository defines the data access contract for the notification
// routes of the acting workspace.
type NotificationRouteRepository interface {
	List(ctx context.Context) ([]model.NotificationRoute, error)
	Get(ctx context.Context, id string) (*model.NotificationRoute, error)
	Create(ctx context.Context, route *model.NotificationRoute) (*model.NotificationRoute, error)
	Update(
		ctx context.Context,
		route *model.NotificationRoute,
		read time.Time,
	) (*model.NotificationRoute, error)
	Delete(ctx context.Context, id string) error
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

// List answers every notification route of the acting workspace, in the order of the
// identifiers.
func (r *NotificationRoute) List(ctx context.Context) ([]model.NotificationRoute, error) {
	routes := []model.NotificationRoute{}

	query := `SELECT ` + notificationRouteColumns + ` FROM public.notification_routes ORDER BY id;`

	if err := r.tx.SelectContext(ctx, &routes, query); err != nil {
		return nil, fmt.Errorf("listing NotificationRoutes: %w", err)
	}

	return routes, nil
}

// Get answers one notification route of the acting workspace.
func (r *NotificationRoute) Get(ctx context.Context, id string) (*model.NotificationRoute, error) {
	var route model.NotificationRoute

	query := `SELECT ` + notificationRouteColumns + ` FROM public.notification_routes WHERE id = $1;`

	err := r.tx.GetContext(ctx, &route, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("reading a NotificationRoute: %w", errs.ErrNotificationRouteNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("reading a NotificationRoute: %w", err)
	}

	return &route, nil
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
		return nil, fmt.Errorf("inserting NotificationRoute: %w", routeFailure(err))
	}

	return &stored, nil
}

// Update writes the destination and the enabled state while the row still carries the
// moment that the caller read, and answers precondition.ErrModified when it moved since.
func (r *NotificationRoute) Update(
	ctx context.Context,
	route *model.NotificationRoute,
	read time.Time,
) (*model.NotificationRoute, error) {
	var stored model.NotificationRoute

	query := `
		UPDATE public.notification_routes SET destination = $2, enabled = $3
		WHERE id = $1 AND updated_at = $4
		RETURNING ` + notificationRouteColumns + `;`

	err := r.tx.GetContext(ctx, &stored, query, route.ID, route.Destination, route.Enabled, read)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("changing a NotificationRoute: %w", precondition.ErrModified)
	}

	if err != nil {
		return nil, fmt.Errorf("changing a NotificationRoute: %w", routeFailure(err))
	}

	return &stored, nil
}

// Delete removes a notification route of the acting workspace.
func (r *NotificationRoute) Delete(ctx context.Context, id string) error {
	result, err := r.tx.ExecContext(
		ctx,
		`DELETE FROM public.notification_routes WHERE id = $1;`,
		id,
	)
	if err != nil {
		return fmt.Errorf("deleting a NotificationRoute: %w", err)
	}

	return requireOneRow(result, "deleting a NotificationRoute", errs.ErrNotificationRouteNotFound)
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

// routeFailure names the two constraints that a write of a route can break. It keeps
// the error of the database in the chain, so the log still names the constraint.
func routeFailure(err error) error {
	const foreignKeyViolation = "23503"

	if isUniqueViolation(err, notificationRouteKey) {
		return fmt.Errorf("%w: %w", errs.ErrNotificationRouteExists, err)
	}

	var pgErr pgx.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation &&
		pgErr.ConstraintName == notificationRouteProviderKey {
		return fmt.Errorf("%w: %w", errs.ErrNotificationProviderNotFound, err)
	}

	return err
}
