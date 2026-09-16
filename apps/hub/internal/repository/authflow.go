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

const authFlowColumns = `id, state, intent, user_id, invitation_id, nonce, verifier,
	redirect, provider, expires_at, consumed_at, created_at`

// AuthFlowRepository defines the data access contract for an authorization flow.
type AuthFlowRepository interface {
	Create(ctx context.Context, flow model.AuthFlow) (*model.AuthFlow, error)
	ConsumeByState(ctx context.Context, state string) (*model.AuthFlow, error)
}

// AuthFlow is a SQL based implementation of AuthFlowRepository.
type AuthFlow struct {
	db *sqlx.DB
}

var _ AuthFlowRepository = (*AuthFlow)(nil)

// NewAuthFlow creates a new AuthFlow repository instance.
func NewAuthFlow(db *sqlx.DB) *AuthFlow {
	return &AuthFlow{db: db}
}

// Create writes one authorization flow.
func (r *AuthFlow) Create(ctx context.Context, flow model.AuthFlow) (*model.AuthFlow, error) {
	var entity model.AuthFlow

	query := `
		INSERT INTO public.auth_flows
			(state, intent, user_id, invitation_id, nonce, verifier, redirect, provider, expires_at)
		VALUES (:state, :intent, :user_id, :invitation_id, :nonce, :verifier, :redirect,
		        :provider, :expires_at)
		RETURNING ` + authFlowColumns + `;`

	statement, args, err := sqlx.Named(query, flow)
	if err != nil {
		return nil, fmt.Errorf("binding AuthFlow arguments: %w", err)
	}

	if err = r.db.GetContext(ctx, &entity, r.db.Rebind(statement), args...); err != nil {
		return nil, fmt.Errorf("creating an AuthFlow: %w", err)
	}

	return &entity, nil
}

// ConsumeByState spends the flow that the state names and returns it.
func (r *AuthFlow) ConsumeByState(ctx context.Context, state string) (*model.AuthFlow, error) {
	var entity model.AuthFlow

	query := `
		UPDATE public.auth_flows
		SET consumed_at = NOW()
		WHERE state = $1 AND consumed_at IS NULL
		RETURNING ` + authFlowColumns + `;`

	if err := r.db.GetContext(ctx, &entity, query, state); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("consuming an AuthFlow: %w", errs.ErrAuthFlowNotFound)
		}

		return nil, fmt.Errorf("consuming an AuthFlow: %w", err)
	}

	return &entity, nil
}
