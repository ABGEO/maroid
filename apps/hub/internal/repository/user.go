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

const (
	userColumns = `id, first_name, last_name, status, created_at, updated_at`

	// userColumnsOfU repeats userColumns for a join, where a bare name is
	// ambiguous. The two lists change together.
	userColumnsOfU = `u.id, u.first_name, u.last_name, u.status, u.created_at, u.updated_at`
)

// UserRepository defines the data access contract for the user record.
type UserRepository interface {
	GetActiveByID(ctx context.Context, id string) (*model.User, error)
	ListActive(ctx context.Context) ([]model.User, error)
	Create(ctx context.Context, tx *sqlx.Tx, firstName string, lastName string) (*model.User, error)
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

// ListActive retrieves every active user record, oldest first.
func (r *User) ListActive(ctx context.Context) ([]model.User, error) {
	var entities []model.User

	query := `SELECT ` + userColumns + ` FROM public.users WHERE status = $1 ORDER BY id;`

	if err := r.db.SelectContext(ctx, &entities, query, model.StatusActive); err != nil {
		return nil, fmt.Errorf("listing active Users: %w", err)
	}

	return entities, nil
}

// Create writes one user record.
func (r *User) Create(
	ctx context.Context,
	tx *sqlx.Tx,
	firstName string,
	lastName string,
) (*model.User, error) {
	var entity model.User

	query := `
		INSERT INTO public.users (first_name, last_name)
		VALUES (NULLIF($1, ''), NULLIF($2, ''))
		RETURNING ` + userColumns + `;`

	if err := tx.GetContext(ctx, &entity, query, firstName, lastName); err != nil {
		return nil, fmt.Errorf("creating a User: %w", err)
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
