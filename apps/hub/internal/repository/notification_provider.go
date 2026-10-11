package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/libs/rest/precondition"
)

const notificationProviderColumns = `id, workspace_id, kind, name, fields, enabled, created_at, updated_at`

// NotificationProviderRepository defines the data access contract for the
// notification providers of the acting workspace.
type NotificationProviderRepository interface {
	List(ctx context.Context) ([]model.NotificationProvider, error)
	Get(ctx context.Context, id string) (*model.NotificationProvider, error)
	Create(
		ctx context.Context,
		provider *model.NotificationProvider,
	) (*model.NotificationProvider, error)
	Update(
		ctx context.Context,
		provider *model.NotificationProvider,
		read time.Time,
	) (*model.NotificationProvider, error)
	Delete(ctx context.Context, id string) error
}

// NotificationProvider is a SQL based implementation of NotificationProviderRepository.
type NotificationProvider struct {
	tx *sqlx.Tx
}

var _ NotificationProviderRepository = (*NotificationProvider)(nil)

// NewNotificationProvider creates a new NotificationProvider repository instance.
func NewNotificationProvider(tx *sqlx.Tx) *NotificationProvider {
	return &NotificationProvider{tx: tx}
}

// List answers every notification provider of the acting workspace, the oldest first.
func (r *NotificationProvider) List(ctx context.Context) ([]model.NotificationProvider, error) {
	providers := []model.NotificationProvider{}

	query := `SELECT ` + notificationProviderColumns + `
		FROM public.notification_providers ORDER BY created_at, id;`

	if err := r.tx.SelectContext(ctx, &providers, query); err != nil {
		return nil, fmt.Errorf("listing NotificationProviders: %w", err)
	}

	return providers, nil
}

// Get answers one notification provider of the acting workspace.
func (r *NotificationProvider) Get(
	ctx context.Context,
	id string,
) (*model.NotificationProvider, error) {
	var provider model.NotificationProvider

	query := `SELECT ` + notificationProviderColumns + `
		FROM public.notification_providers WHERE id = $1;`

	err := r.tx.GetContext(ctx, &provider, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(
			"reading a NotificationProvider: %w",
			errs.ErrNotificationProviderNotFound,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("reading a NotificationProvider: %w", err)
	}

	return &provider, nil
}

// Create stores a notification provider in the acting workspace. It reads the kind,
// the name, the fields, and the enabled state, and answers the stored row.
func (r *NotificationProvider) Create(
	ctx context.Context,
	provider *model.NotificationProvider,
) (*model.NotificationProvider, error) {
	var stored model.NotificationProvider

	query := `
		INSERT INTO public.notification_providers (kind, name, fields, enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + notificationProviderColumns + `;`

	err := r.tx.GetContext(
		ctx, &stored, query, provider.Kind, provider.Name, provider.Fields, provider.Enabled,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting NotificationProvider: %w", err)
	}

	return &stored, nil
}

// Update writes the name, the fields, and the enabled state while the row still
// carries the moment that the caller read, and answers precondition.ErrModified when
// the row moved since.
func (r *NotificationProvider) Update(
	ctx context.Context,
	provider *model.NotificationProvider,
	read time.Time,
) (*model.NotificationProvider, error) {
	var stored model.NotificationProvider

	query := `
		UPDATE public.notification_providers SET name = $2, fields = $3, enabled = $4
		WHERE id = $1 AND updated_at = $5
		RETURNING ` + notificationProviderColumns + `;`

	err := r.tx.GetContext(
		ctx, &stored, query, provider.ID, provider.Name, provider.Fields, provider.Enabled, read,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("changing a NotificationProvider: %w", precondition.ErrModified)
	}

	if err != nil {
		return nil, fmt.Errorf("changing a NotificationProvider: %w", err)
	}

	return &stored, nil
}

// Delete removes a notification provider of the acting workspace. Its routes go with it.
func (r *NotificationProvider) Delete(ctx context.Context, id string) error {
	result, err := r.tx.ExecContext(
		ctx,
		`DELETE FROM public.notification_providers WHERE id = $1;`,
		id,
	)
	if err != nil {
		return fmt.Errorf("deleting a NotificationProvider: %w", err)
	}

	return requireOneRow(
		result,
		"deleting a NotificationProvider",
		errs.ErrNotificationProviderNotFound,
	)
}

// requireOneRow answers notFound when the statement touched no row.
func requireOneRow(result sql.Result, doing string, notFound error) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", doing, err)
	}

	if affected == 0 {
		return fmt.Errorf("%s: %w", doing, notFound)
	}

	return nil
}
