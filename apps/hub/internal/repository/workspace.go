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
	workspaceColumns    = `id, name, created_at, updated_at`
	workspaceColumnsOfW = `w.id, w.name, w.created_at, w.updated_at`
)

// WorkspaceRepository defines the data access contract for a workspace.
type WorkspaceRepository interface {
	Create(ctx context.Context, tx *sqlx.Tx, name string) (*model.Workspace, error)
	GetByID(ctx context.Context, id string) (*model.Workspace, error)
	ListOfUser(ctx context.Context, userID string) ([]model.Workspace, error)
}

// Workspace is a SQL based implementation of WorkspaceRepository.
type Workspace struct {
	db *sqlx.DB
}

var _ WorkspaceRepository = (*Workspace)(nil)

// NewWorkspace creates a new Workspace repository instance.
func NewWorkspace(db *sqlx.DB) *Workspace {
	return &Workspace{db: db}
}

// Create writes one workspace.
func (r *Workspace) Create(
	ctx context.Context,
	tx *sqlx.Tx,
	name string,
) (*model.Workspace, error) {
	var entity model.Workspace

	query := `INSERT INTO public.workspaces (name) VALUES ($1) RETURNING ` + workspaceColumns + `;`

	if err := tx.GetContext(ctx, &entity, query, name); err != nil {
		return nil, fmt.Errorf("creating a Workspace: %w", err)
	}

	return &entity, nil
}

// GetByID retrieves the workspace with the given identifier.
func (r *Workspace) GetByID(ctx context.Context, id string) (*model.Workspace, error) {
	var entity model.Workspace

	query := `SELECT ` + workspaceColumns + ` FROM public.workspaces WHERE id = $1;`

	if err := r.db.GetContext(ctx, &entity, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("getting a Workspace by ID: %w", errs.ErrWorkspaceNotFound)
		}

		return nil, fmt.Errorf("getting a Workspace by ID: %w", err)
	}

	return &entity, nil
}

// ListOfUser retrieves every workspace that the user record is a member of,
// ordered by the name, then by the identifier.
func (r *Workspace) ListOfUser(ctx context.Context, userID string) ([]model.Workspace, error) {
	entities := []model.Workspace{}

	query := `
		SELECT ` + workspaceColumnsOfW + `
		FROM public.workspaces w
		JOIN public.workspace_members m ON m.workspace_id = w.id
		WHERE m.user_id = $1
		ORDER BY w.name, w.id;`

	if err := r.db.SelectContext(ctx, &entities, query, userID); err != nil {
		return nil, fmt.Errorf("listing the Workspaces of a User: %w", err)
	}

	return entities, nil
}
