package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/model"
)

const notificationProviderColumns = `id, workspace_id, kind, name, fields, enabled, created_at, updated_at`

// NotificationProviderRepository defines the data access contract for the
// notification providers of the acting workspace.
type NotificationProviderRepository interface {
	Create(
		ctx context.Context,
		provider *model.NotificationProvider,
	) (*model.NotificationProvider, error)
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
