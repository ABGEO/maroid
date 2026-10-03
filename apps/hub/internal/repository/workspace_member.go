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
	"github.com/abgeo/maroid/libs/rest/precondition"
)

const (
	workspaceMemberKey = "workspace_members_pkey"
	memberColumnsOfM   = `m.workspace_id, m.user_id, m.role, u.first_name, u.last_name, m.created_at, m.updated_at`
)

// WorkspaceMemberRepository defines the data access contract for a membership.
type WorkspaceMemberRepository interface {
	Add(
		ctx context.Context,
		tx *sqlx.Tx,
		workspaceID string,
		userID string,
		role pluginapi.Role,
	) (*model.Member, error)
	Get(ctx context.Context, workspaceID string, userID string) (*model.Member, error)
	List(ctx context.Context, workspaceID string) ([]model.Member, error)
	Remove(ctx context.Context, tx *sqlx.Tx, workspaceID string, userID string) error
	ChangeRole(
		ctx context.Context,
		tx *sqlx.Tx,
		workspaceID string,
		userID string,
		role pluginapi.Role,
		ifMatch *time.Time,
	) (*model.Member, error)
	CountManagers(ctx context.Context, tx *sqlx.Tx, workspaceID string) (int, error)
	Candidates(ctx context.Context, workspaceID string) ([]model.User, error)
}

// WorkspaceMember is a SQL based implementation of WorkspaceMemberRepository.
type WorkspaceMember struct {
	db *sqlx.DB
}

var _ WorkspaceMemberRepository = (*WorkspaceMember)(nil)

// NewWorkspaceMember creates a new WorkspaceMember repository instance.
func NewWorkspaceMember(db *sqlx.DB) *WorkspaceMember {
	return &WorkspaceMember{db: db}
}

// Add writes one membership with its role, and answers it with the names of the record.
func (r *WorkspaceMember) Add(
	ctx context.Context,
	tx *sqlx.Tx,
	workspaceID string,
	userID string,
	role pluginapi.Role,
) (*model.Member, error) {
	var entity model.Member

	query := `
		WITH m AS (
			INSERT INTO public.workspace_members (workspace_id, user_id, role)
			VALUES ($1, $2, $3)
			RETURNING workspace_id, user_id, role, created_at, updated_at
		)
		SELECT ` + memberColumnsOfM + `
		FROM m
		JOIN public.users u ON u.id = m.user_id;`

	if err := tx.GetContext(ctx, &entity, query, workspaceID, userID, role); err != nil {
		if isUniqueViolation(err, workspaceMemberKey) {
			return nil, fmt.Errorf("adding a Member: %w", errs.ErrMemberExists)
		}

		return nil, fmt.Errorf("adding a Member: %w", err)
	}

	return &entity, nil
}

// Get retrieves the membership of the user record in the workspace.
func (r *WorkspaceMember) Get(
	ctx context.Context,
	workspaceID string,
	userID string,
) (*model.Member, error) {
	var entity model.Member

	query := `
		SELECT ` + memberColumnsOfM + `
		FROM public.workspace_members m
		JOIN public.users u ON u.id = m.user_id
		WHERE m.workspace_id = $1 AND m.user_id = $2;`

	if err := r.db.GetContext(ctx, &entity, query, workspaceID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("getting a Member: %w", errs.ErrMemberNotFound)
		}

		return nil, fmt.Errorf("getting a Member: %w", err)
	}

	return &entity, nil
}

// List retrieves every member of the workspace, in the order of joining, then by
// the user identifier, because the table carries no id to sort by.
func (r *WorkspaceMember) List(ctx context.Context, workspaceID string) ([]model.Member, error) {
	entities := []model.Member{}

	query := `
		SELECT ` + memberColumnsOfM + `
		FROM public.workspace_members m
		JOIN public.users u ON u.id = m.user_id
		WHERE m.workspace_id = $1
		ORDER BY m.created_at, m.user_id;`

	if err := r.db.SelectContext(ctx, &entities, query, workspaceID); err != nil {
		return nil, fmt.Errorf("listing the Members of a Workspace: %w", err)
	}

	return entities, nil
}

// Remove deletes the membership of the user record in the workspace. Every record
// of the workspace stays, because no row names a membership.
func (r *WorkspaceMember) Remove(
	ctx context.Context,
	tx *sqlx.Tx,
	workspaceID string,
	userID string,
) error {
	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM public.workspace_members WHERE workspace_id = $1 AND user_id = $2;`,
		workspaceID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("removing a Member: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("removing a Member: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("removing a Member: %w", errs.ErrMemberNotFound)
	}

	return nil
}

// Candidates retrieves every active user record that is no member of the
// workspace, ordered by the last name, then by the first name.
func (r *WorkspaceMember) Candidates(
	ctx context.Context,
	workspaceID string,
) ([]model.User, error) {
	entities := []model.User{}

	query := `
		SELECT ` + userColumnsOfU + `
		FROM public.users u
		WHERE u.status = $2
		  AND NOT EXISTS (
			SELECT 1 FROM public.workspace_members m
			WHERE m.workspace_id = $1 AND m.user_id = u.id
		  )
		ORDER BY u.last_name NULLS LAST, u.first_name NULLS LAST, u.id;`

	if err := r.db.SelectContext(
		ctx,
		&entities,
		query,
		workspaceID,
		model.StatusActive,
	); err != nil {
		return nil, fmt.Errorf("listing the candidates of a Workspace: %w", err)
	}

	return entities, nil
}

// ChangeRole sets the role of the membership, and answers it with the names of the
// record. With a validator it changes the row only while the row still carries it.
func (r *WorkspaceMember) ChangeRole(
	ctx context.Context,
	tx *sqlx.Tx,
	workspaceID string,
	userID string,
	role pluginapi.Role,
	ifMatch *time.Time,
) (*model.Member, error) {
	var entity model.Member

	query := `
		WITH m AS (
			UPDATE public.workspace_members SET role = $3
			WHERE workspace_id = $1 AND user_id = $2
				AND ($4::timestamptz IS NULL OR updated_at = $4)
			RETURNING workspace_id, user_id, role, created_at, updated_at
		)
		SELECT ` + memberColumnsOfM + `
		FROM m
		JOIN public.users u ON u.id = m.user_id;`

	err := tx.GetContext(ctx, &entity, query, workspaceID, userID, role, ifMatch)
	if errors.Is(err, sql.ErrNoRows) {
		if ifMatch != nil {
			return nil, fmt.Errorf("changing the role of a Member: %w", precondition.ErrModified)
		}

		return nil, fmt.Errorf("changing the role of a Member: %w", errs.ErrMemberNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("changing the role of a Member: %w", err)
	}

	return &entity, nil
}

// CountManagers counts the managers of the workspace, as the transaction sees them.
func (r *WorkspaceMember) CountManagers(
	ctx context.Context,
	tx *sqlx.Tx,
	workspaceID string,
) (int, error) {
	var count int

	err := tx.GetContext(ctx, &count,
		`SELECT count(*) FROM public.workspace_members WHERE workspace_id = $1 AND role = $2;`,
		workspaceID, pluginapi.RoleManager)
	if err != nil {
		return 0, fmt.Errorf("counting the managers of a Workspace: %w", err)
	}

	return count, nil
}
