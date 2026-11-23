package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/pulse/model"
)

const selectSettings = `
	SELECT id, enabled, seeded_at, paused_until, created_at, updated_at FROM settings`

// SettingsRepository defines the data access contract for Settings entities.
type SettingsRepository interface {
	Get(ctx context.Context) (*model.Settings, error)
	Lock(ctx context.Context) (*model.Settings, error)
	Insert(ctx context.Context) (*model.Settings, error)
	Update(ctx context.Context, entity *model.Settings, ifMatch *time.Time) error
}

// Settings reads and writes the one row of settings of the acting user.
type Settings struct {
	tx *sqlx.Tx
}

var _ SettingsRepository = (*Settings)(nil)

// NewSettings creates a Settings repository over the transaction.
func NewSettings(tx *sqlx.Tx) *Settings {
	return &Settings{tx: tx}
}

// Get answers the row, or sql.ErrNoRows when the person never wrote one.
func (r *Settings) Get(ctx context.Context) (*model.Settings, error) {
	var entity model.Settings

	if err := r.tx.GetContext(ctx, &entity, selectSettings+";"); err != nil {
		return nil, fmt.Errorf("getting Settings: %w", err)
	}

	return &entity, nil
}

// Lock answers the row and holds it until the transaction ends, or sql.ErrNoRows.
func (r *Settings) Lock(ctx context.Context) (*model.Settings, error) {
	var entity model.Settings

	err := r.tx.GetContext(ctx, &entity, selectSettings+" FOR UPDATE;")
	if err != nil {
		return nil, fmt.Errorf("locking Settings: %w", err)
	}

	return &entity, nil
}

// Insert stores the first row of the person, and answers the row that the person
// holds when another request stored it first.
func (r *Settings) Insert(ctx context.Context) (*model.Settings, error) {
	var entity model.Settings

	err := r.tx.GetContext(ctx, &entity, `
		INSERT INTO settings DEFAULT VALUES
		ON CONFLICT ON CONSTRAINT settings_user_key DO NOTHING
		RETURNING id, enabled, seeded_at, paused_until, created_at, updated_at;`)
	if errors.Is(err, sql.ErrNoRows) {
		return r.Lock(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("inserting Settings: %w", err)
	}

	return &entity, nil
}

// Update writes the row under the validator and refreshes the entity.
func (r *Settings) Update(ctx context.Context, entity *model.Settings, ifMatch *time.Time) error {
	err := getNamed(ctx, r.tx, entity, `
		UPDATE settings
		SET enabled = :enabled, seeded_at = :seeded_at, paused_until = :paused_until
		WHERE id = :id
		  AND (CAST(:if_match AS TIMESTAMPTZ) IS NULL OR updated_at = CAST(:if_match AS TIMESTAMPTZ))
		RETURNING id, enabled, seeded_at, paused_until, created_at, updated_at;`,
		map[string]any{
			paramID:        entity.ID,
			"enabled":      entity.Enabled,
			"seeded_at":    entity.SeededAt,
			"paused_until": entity.PausedUntil,
			paramIfMatch:   ifMatch,
		},
	)
	if errors.Is(err, sql.ErrNoRows) {
		return missed(ifMatch)
	}

	if err != nil {
		return fmt.Errorf("updating Settings: %w", err)
	}

	return nil
}
