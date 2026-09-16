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
)

const invitationColumns = `id, user_id, token_hash, expires_at, consumed_at, created_at`

// InvitationRepository defines the data access contract for an invitation.
type InvitationRepository interface {
	Create(
		ctx context.Context,
		tx *sqlx.Tx,
		userID string,
		tokenHash []byte,
		expiresAt time.Time,
	) (*model.Invitation, error)
	GetValidByTokenHash(ctx context.Context, tokenHash []byte) (*model.Invitation, error)
	Consume(ctx context.Context, tx *sqlx.Tx, id string) (*model.Invitation, error)
}

// Invitation is a SQL based implementation of InvitationRepository.
type Invitation struct {
	db *sqlx.DB
}

var _ InvitationRepository = (*Invitation)(nil)

// NewInvitation creates a new Invitation repository instance.
func NewInvitation(db *sqlx.DB) *Invitation {
	return &Invitation{db: db}
}

// Create writes one invitation for the user record.
func (r *Invitation) Create(
	ctx context.Context,
	tx *sqlx.Tx,
	userID string,
	tokenHash []byte,
	expiresAt time.Time,
) (*model.Invitation, error) {
	var entity model.Invitation

	query := `
		INSERT INTO public.invitations (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING ` + invitationColumns + `;`

	if err := tx.GetContext(ctx, &entity, query, userID, tokenHash, expiresAt); err != nil {
		return nil, fmt.Errorf("creating an Invitation: %w", err)
	}

	return &entity, nil
}

// GetValidByTokenHash retrieves the invitation that the digest names, when that
// invitation is neither consumed nor expired.
func (r *Invitation) GetValidByTokenHash(
	ctx context.Context,
	tokenHash []byte,
) (*model.Invitation, error) {
	var entity model.Invitation

	query := `
		SELECT ` + invitationColumns + `
		FROM public.invitations
		WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > NOW();`

	if err := r.db.GetContext(ctx, &entity, query, tokenHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("getting a valid Invitation: %w", errs.ErrInvitationNotValid)
		}

		return nil, fmt.Errorf("getting a valid Invitation: %w", err)
	}

	return &entity, nil
}

// Consume spends the invitation.
func (r *Invitation) Consume(
	ctx context.Context,
	tx *sqlx.Tx,
	id string,
) (*model.Invitation, error) {
	var entity model.Invitation

	query := `
		UPDATE public.invitations
		SET consumed_at = NOW()
		WHERE id = $1 AND consumed_at IS NULL AND expires_at > NOW()
		RETURNING ` + invitationColumns + `;`

	if err := tx.GetContext(ctx, &entity, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("consuming an Invitation: %w", errs.ErrInvitationNotValid)
		}

		return nil, fmt.Errorf("consuming an Invitation: %w", err)
	}

	return &entity, nil
}
