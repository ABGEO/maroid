// Package repository holds the data access of the hub.
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

const (
	userColumns = `id, first_name, last_name, status, is_administrator, created_at, updated_at`

	// userColumnsOfU repeats userColumns for a join, where a bare name is
	// ambiguous. The two lists change together.
	userColumnsOfU = `u.id, u.first_name, u.last_name, u.status, u.is_administrator, ` +
		`u.created_at, u.updated_at`
)

// UserRepository defines the data access contract for the user record.
type UserRepository interface {
	GetActiveByID(ctx context.Context, id string) (*model.User, error)
	GetByID(ctx context.Context, id string) (*model.User, error)
	List(ctx context.Context) ([]model.User, error)
	ListActive(ctx context.Context) ([]model.User, error)
	Create(ctx context.Context, tx *sqlx.Tx, firstName string, lastName string) (*model.User, error)
	Change(ctx context.Context, tx *sqlx.Tx, id string, change UserChange) (*model.User, error)
	LockAdministrators(ctx context.Context, tx *sqlx.Tx) error
	CountActiveAdministrators(ctx context.Context, tx *sqlx.Tx) (int, error)
}

// UserChange holds the columns that one change of a user record writes. A nil
// member keeps its column. A non-nil IfMatch changes the row only while the row
// still carries it.
type UserChange struct {
	Status        *model.Status
	Administrator *bool
	IfMatch       *time.Time
}

// User is a SQL based implementation of UserRepository.
//
// It holds the pool and not a transaction, because each operation is one
// statement on the request path. The repository of a plugin binds a transaction
// instead, because it reaches its data through the search path that PluginDB sets.
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
// A person that holds no active record reaches nothing, so a blocked record
// answers the same way as a record that does not exist.
func (r *User) GetActiveByID(ctx context.Context, id string) (*model.User, error) {
	query := `SELECT ` + userColumns + ` FROM public.users WHERE id = $1 AND status = $2;`

	return r.get(ctx, "ID", query, id, model.StatusActive)
}

// GetByID retrieves the user record with the given identifier, active or blocked.
func (r *User) GetByID(ctx context.Context, id string) (*model.User, error) {
	query := `SELECT ` + userColumns + ` FROM public.users WHERE id = $1;`

	return r.get(ctx, "ID", query, id)
}

// List retrieves every user record, ordered by the last name, then by the first name.
func (r *User) List(ctx context.Context) ([]model.User, error) {
	entities := []model.User{}

	query := `SELECT ` + userColumns + ` FROM public.users
		ORDER BY last_name NULLS LAST, first_name NULLS LAST, id;`

	if err := r.db.SelectContext(ctx, &entities, query); err != nil {
		return nil, fmt.Errorf("listing Users: %w", err)
	}

	return entities, nil
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

// Change writes the status and the mark of a user record, and answers the record.
func (r *User) Change(
	ctx context.Context,
	tx *sqlx.Tx,
	id string,
	change UserChange,
) (*model.User, error) {
	var entity model.User

	query := `
		UPDATE public.users
		SET status = COALESCE($2, status), is_administrator = COALESCE($3, is_administrator)
		WHERE id = $1 AND ($4::timestamptz IS NULL OR updated_at = $4)
		RETURNING ` + userColumns + `;`

	err := tx.GetContext(
		ctx,
		&entity,
		query,
		id,
		change.Status,
		change.Administrator,
		change.IfMatch,
	)
	if errors.Is(err, sql.ErrNoRows) {
		if change.IfMatch != nil {
			return nil, fmt.Errorf("changing a User: %w", precondition.ErrModified)
		}

		return nil, fmt.Errorf("changing a User: %w", errs.ErrUserNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("changing a User: %w", err)
	}

	return &entity, nil
}

// LockAdministrators holds every active administrator until the transaction ends,
// so two changes that each count the administrators run one after the other.
func (r *User) LockAdministrators(ctx context.Context, tx *sqlx.Tx) error {
	_, err := tx.ExecContext(ctx, `
		SELECT id FROM public.users
		WHERE is_administrator AND status = $1
		FOR UPDATE;`, model.StatusActive)
	if err != nil {
		return fmt.Errorf("locking the administrators: %w", err)
	}

	return nil
}

// CountActiveAdministrators counts the active administrators, as the transaction sees
// them.
func (r *User) CountActiveAdministrators(ctx context.Context, tx *sqlx.Tx) (int, error) {
	var count int

	err := tx.GetContext(ctx, &count,
		`SELECT count(*) FROM public.users WHERE is_administrator AND status = $1;`,
		model.StatusActive)
	if err != nil {
		return 0, fmt.Errorf("counting the administrators: %w", err)
	}

	return count, nil
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
