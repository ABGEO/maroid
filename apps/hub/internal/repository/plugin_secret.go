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
	"github.com/abgeo/maroid/libs/pluginapi"
)

//nolint:gosec // the names of the columns, not a credential.
const pluginSecretColumns = `id, plugin_id, name, ciphertext, expires_at, created_at, updated_at`

// PluginSecretRepository defines the data access contract for the plugin secrets of a
// user and of a workspace. The scope selects the table.
type PluginSecretRepository interface {
	Get(
		ctx context.Context,
		scope pluginapi.SecretScope,
		pluginID string,
		name string,
	) (*model.PluginSecret, error)
	Upsert(
		ctx context.Context,
		scope pluginapi.SecretScope,
		pluginID string,
		name string,
		ciphertext string,
		expiresAt *time.Time,
	) error
	Delete(ctx context.Context, scope pluginapi.SecretScope, pluginID string, name string) error
	DeleteExpired(ctx context.Context, scope pluginapi.SecretScope) (int64, error)
}

// PluginSecret is a SQL based implementation of PluginSecretRepository.
type PluginSecret struct {
	tx *sqlx.Tx
}

var _ PluginSecretRepository = (*PluginSecret)(nil)

// NewPluginSecret creates a new PluginSecret repository instance.
func NewPluginSecret(tx *sqlx.Tx) *PluginSecret {
	return &PluginSecret{tx: tx}
}

// Get retrieves the plugin secret of the scope under the name. It returns a nil
// entity when no row holds the name, or when the row expired.
func (r *PluginSecret) Get(
	ctx context.Context,
	scope pluginapi.SecretScope,
	pluginID string,
	name string,
) (*model.PluginSecret, error) {
	table, err := secretTableOf(scope)
	if err != nil {
		return nil, err
	}

	query := `SELECT ` + pluginSecretColumns + ` FROM ` + table.name + `
		WHERE plugin_id = $1 AND name = $2 AND (expires_at IS NULL OR expires_at > now());`

	var entity model.PluginSecret

	if err = r.tx.GetContext(ctx, &entity, query, pluginID, name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // no row is an absent plugin secret, not a failure.
		}

		return nil, fmt.Errorf("getting the PluginSecret of the %s: %w", scope, err)
	}

	return &entity, nil
}

// Upsert stores the ciphertext of the scope under the name. A row that holds the name
// takes the new ciphertext and the new expiry.
func (r *PluginSecret) Upsert(
	ctx context.Context,
	scope pluginapi.SecretScope,
	pluginID string,
	name string,
	ciphertext string,
	expiresAt *time.Time,
) error {
	table, err := secretTableOf(scope)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO ` + table.name + ` (plugin_id, name, ciphertext, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (` + table.owner + `, plugin_id, name)
		DO UPDATE SET ciphertext = EXCLUDED.ciphertext, expires_at = EXCLUDED.expires_at;`

	if _, err = r.tx.ExecContext(ctx, query, pluginID, name, ciphertext, expiresAt); err != nil {
		return fmt.Errorf("upserting the PluginSecret of the %s: %w", scope, err)
	}

	return nil
}

// Delete removes the plugin secret of the scope under the name. A name that the scope
// does not hold is no failure.
func (r *PluginSecret) Delete(
	ctx context.Context,
	scope pluginapi.SecretScope,
	pluginID string,
	name string,
) error {
	table, err := secretTableOf(scope)
	if err != nil {
		return err
	}

	query := `DELETE FROM ` + table.name + ` WHERE plugin_id = $1 AND name = $2;`

	if _, err = r.tx.ExecContext(ctx, query, pluginID, name); err != nil {
		return fmt.Errorf("deleting the PluginSecret of the %s: %w", scope, err)
	}

	return nil
}

// DeleteExpired removes every plugin secret of the scope whose expiry passed, and
// answers the count of the removed rows.
func (r *PluginSecret) DeleteExpired(
	ctx context.Context,
	scope pluginapi.SecretScope,
) (int64, error) {
	table, err := secretTableOf(scope)
	if err != nil {
		return 0, err
	}

	result, err := r.tx.ExecContext(ctx, `DELETE FROM `+table.name+` WHERE expires_at <= now();`)
	if err != nil {
		return 0, fmt.Errorf("removing the expired PluginSecrets of the %s: %w", scope, err)
	}

	removed, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("counting the removed PluginSecrets of the %s: %w", scope, err)
	}

	return removed, nil
}

func secretTableOf(scope pluginapi.SecretScope) (scopedTable, error) {
	switch scope {
	case pluginapi.SecretScopeUser:
		return scopedTable{name: "public.plugin_user_secrets", owner: "user_id"}, nil
	case pluginapi.SecretScopeWorkspace:
		return scopedTable{name: "public.plugin_workspace_secrets", owner: "workspace_id"}, nil
	default:
		return scopedTable{}, fmt.Errorf("%w: %q", errs.ErrUnknownSecretScope, scope)
	}
}
