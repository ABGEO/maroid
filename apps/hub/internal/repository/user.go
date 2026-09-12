// Package repository holds the data access of the hub.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
)

const userColumns = `id, telegram_id, username, display_name, picture_url, status, created_at, updated_at`

// UserRepository defines the data access contract for the user record.
type UserRepository interface {
	GetActiveByID(ctx context.Context, id string) (*model.User, error)
	GetActiveByTelegramID(ctx context.Context, telegramID int64) (*model.User, error)
	ListActive(ctx context.Context) ([]model.User, error)
	SyncProfileByTelegramID(
		ctx context.Context,
		telegramID int64,
		profile model.Profile,
	) (*model.User, error)
}

// User is a SQL based implementation of UserRepository.
//
// It holds the pool and not a transaction, because each operation is one
// statement on the request path. `REP-003` binds the repository of a plugin,
// which reaches its data through the search path that PluginDB sets.
type User struct {
	db *sqlx.DB
}

var _ UserRepository = (*User)(nil)

// NewUser creates a new User repository instance.
func NewUser(db *sqlx.DB) *User {
	return &User{db: db}
}

// GetActiveByID retrieves the active user record with the given identifier.
//
// IDENT-FR-002: A person that holds no active record reaches nothing, so a
// blocked record answers the same way as a record that does not exist.
func (r *User) GetActiveByID(ctx context.Context, id string) (*model.User, error) {
	query := `SELECT ` + userColumns + ` FROM public.users WHERE id = $1 AND status = $2;`

	return r.get(ctx, "ID", query, id, model.StatusActive)
}

// GetActiveByTelegramID retrieves the active user record with the given Telegram identifier.
func (r *User) GetActiveByTelegramID(
	ctx context.Context,
	telegramID int64,
) (*model.User, error) {
	query := `SELECT ` + userColumns + ` FROM public.users WHERE telegram_id = $1 AND status = $2;`

	return r.get(ctx, "Telegram ID", query, telegramID, model.StatusActive)
}

// ListActive retrieves every active user record, oldest first.
//
// OWN-009: A cron job that declares CronScopePerUser runs one time for each
// record that this returns.
func (r *User) ListActive(ctx context.Context) ([]model.User, error) {
	var entities []model.User

	query := `SELECT ` + userColumns + ` FROM public.users WHERE status = $1 ORDER BY id;`

	if err := r.db.SelectContext(ctx, &entities, query, model.StatusActive); err != nil {
		return nil, fmt.Errorf("listing active Users: %w", err)
	}

	return entities, nil
}

// SyncProfileByTelegramID writes the Telegram profile onto the active record of
// that person and returns the record.
//
// IDENT-DD-005: The login refreshes the profile. The statement touches no
// identity column, so IDENT-FR-001 keeps the record permanent.
func (r *User) SyncProfileByTelegramID(
	ctx context.Context,
	telegramID int64,
	profile model.Profile,
) (*model.User, error) {
	var entity model.User

	query := `
		UPDATE public.users
		SET username     = NULLIF(:username, ''),
		    display_name = NULLIF(:display_name, ''),
		    picture_url  = NULLIF(:picture_url, '')
		WHERE telegram_id = :telegram_id AND status = :status
		RETURNING ` + userColumns + `;`

	query, args, err := sqlx.Named(query, map[string]any{
		"username":     profile.Username,
		"display_name": profile.DisplayName,
		"picture_url":  profile.PictureURL,
		"telegram_id":  telegramID,
		"status":       model.StatusActive,
	})
	if err != nil {
		return nil, fmt.Errorf("binding User profile arguments: %w", err)
	}

	err = r.db.GetContext(ctx, &entity, r.db.Rebind(query), args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf(
				"syncing the profile of User by Telegram ID: %w",
				errs.ErrUserNotFound,
			)
		}

		return nil, fmt.Errorf("syncing the profile of User by Telegram ID: %w", err)
	}

	return &entity, nil
}

func (r *User) get(
	ctx context.Context,
	by string,
	query string,
	args ...any,
) (*model.User, error) {
	var entity model.User

	if err := r.db.GetContext(ctx, &entity, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("getting active User by %s: %w", by, errs.ErrUserNotFound)
		}

		return nil, fmt.Errorf("getting active User by %s: %w", by, err)
	}

	return &entity, nil
}
