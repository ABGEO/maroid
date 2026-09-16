package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
)

const (
	identityColumns = `id, user_id, provider, provider_user_id, username, display_name,
		picture_url, created_at, updated_at`

	identityProviderUserKey = "identities_provider_user_key"
)

// IdentityRepository defines the data access contract for an identity.
type IdentityRepository interface {
	GetActiveUserByProvider(
		ctx context.Context,
		provider string,
		providerUserID string,
	) (*model.User, error)
	ListByUser(ctx context.Context, userID string) ([]model.Identity, error)
	Attach(
		ctx context.Context,
		tx *sqlx.Tx,
		userID string,
		provider string,
		providerUserID string,
		profile model.Profile,
	) error
	SyncProfile(
		ctx context.Context,
		provider string,
		providerUserID string,
		profile model.Profile,
	) error
	Detach(ctx context.Context, userID string, provider string) error
}

// identityBinding names the parameters of a write. The embedded profile carries
// the three columns that the provider fills.
type identityBinding struct {
	model.Profile

	UserID         string `db:"user_id"`
	Provider       string `db:"provider"`
	ProviderUserID string `db:"provider_user_id"`
}

// Identity is a SQL based implementation of IdentityRepository.
type Identity struct {
	db *sqlx.DB
}

var _ IdentityRepository = (*Identity)(nil)

// NewIdentity creates a new Identity repository instance.
func NewIdentity(db *sqlx.DB) *Identity {
	return &Identity{db: db}
}

// GetActiveUserByProvider retrieves the active user record that the external
// account names.
func (r *Identity) GetActiveUserByProvider(
	ctx context.Context,
	provider string,
	providerUserID string,
) (*model.User, error) {
	var entity model.User

	query := `
		SELECT ` + userColumnsOfU + `
		FROM public.users u
		JOIN public.identities i ON i.user_id = u.id
		WHERE i.provider = $1 AND i.provider_user_id = $2 AND u.status = $3;`

	err := r.db.GetContext(ctx, &entity, query, provider, providerUserID, model.StatusActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf(
				"getting active User by the identity of %s: %w",
				provider,
				errs.ErrUserNotFound,
			)
		}

		return nil, fmt.Errorf("getting active User by the identity of %s: %w", provider, err)
	}

	return &entity, nil
}

// ListByUser retrieves every identity of the user record, oldest first.
func (r *Identity) ListByUser(ctx context.Context, userID string) ([]model.Identity, error) {
	var entities []model.Identity

	query := `SELECT ` + identityColumns +
		` FROM public.identities WHERE user_id = $1 ORDER BY id;`

	if err := r.db.SelectContext(ctx, &entities, query, userID); err != nil {
		return nil, fmt.Errorf("listing the Identities of a User: %w", err)
	}

	return entities, nil
}

// Attach binds the external account to the user record.
func (r *Identity) Attach(
	ctx context.Context,
	tx *sqlx.Tx,
	userID string,
	provider string,
	providerUserID string,
	profile model.Profile,
) error {
	query := `
		INSERT INTO public.identities
			(user_id, provider, provider_user_id, username, display_name, picture_url)
		VALUES (:user_id, :provider, :provider_user_id,
		        NULLIF(:username, ''), NULLIF(:display_name, ''), NULLIF(:picture_url, ''));`

	_, err := tx.NamedExecContext(ctx, query, identityBinding{
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
		Profile:        profile,
	})
	if err != nil {
		if isUniqueViolation(err, identityProviderUserKey) {
			return fmt.Errorf("attaching an Identity: %w", errs.ErrIdentityTaken)
		}

		return fmt.Errorf("attaching an Identity: %w", err)
	}

	return nil
}

// SyncProfile writes the profile that the provider gave onto the identity.
func (r *Identity) SyncProfile(
	ctx context.Context,
	provider string,
	providerUserID string,
	profile model.Profile,
) error {
	query := `
		UPDATE public.identities
		SET username     = NULLIF(:username, ''),
		    display_name = NULLIF(:display_name, ''),
		    picture_url  = NULLIF(:picture_url, '')
		WHERE provider = :provider AND provider_user_id = :provider_user_id;`

	result, err := r.db.NamedExecContext(ctx, query, identityBinding{
		Provider:       provider,
		ProviderUserID: providerUserID,
		Profile:        profile,
	})
	if err != nil {
		return fmt.Errorf("syncing the profile of an Identity: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("syncing the profile of an Identity: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("syncing the profile of an Identity: %w", errs.ErrIdentityNotFound)
	}

	return nil
}

// Detach removes the identity of the provider from the user record.
func (r *Identity) Detach(ctx context.Context, userID string, provider string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("detaching an Identity: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	var locked string

	err = tx.GetContext(
		ctx,
		&locked,
		`SELECT id FROM public.users WHERE id = $1 FOR UPDATE;`,
		userID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("detaching an Identity: %w", errs.ErrUserNotFound)
		}

		return fmt.Errorf("detaching an Identity: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
		DELETE FROM public.identities
		WHERE user_id = $1
		  AND provider = $2
		  AND (SELECT count(*) FROM public.identities WHERE user_id = $1) > 1;`,
		userID, provider,
	)
	if err != nil {
		return fmt.Errorf("detaching an Identity: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("detaching an Identity: %w", err)
	}

	if affected == 0 {
		return r.refuseDetach(ctx, tx, userID, provider)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("detaching an Identity: %w", err)
	}

	return nil
}

// refuseDetach names the reason that the delete removed no row.
func (r *Identity) refuseDetach(
	ctx context.Context,
	tx *sqlx.Tx,
	userID string,
	provider string,
) error {
	var exists bool

	err := tx.GetContext(ctx, &exists, `
		SELECT EXISTS(
			SELECT 1 FROM public.identities WHERE user_id = $1 AND provider = $2
		);`,
		userID, provider,
	)
	if err != nil {
		return fmt.Errorf("detaching an Identity: %w", err)
	}

	if exists {
		return fmt.Errorf("detaching an Identity: %w", errs.ErrLastIdentity)
	}

	return fmt.Errorf("detaching an Identity: %w", errs.ErrIdentityNotFound)
}

// isUniqueViolation reports whether err is the unique violation of the constraint.
func isUniqueViolation(err error, constraint string) bool {
	const uniqueViolation = "23505"

	var pgErr pgx.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == uniqueViolation && pgErr.ConstraintName == constraint
}
