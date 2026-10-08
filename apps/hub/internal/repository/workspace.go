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
	workspaceColumns    = `id, name, created_at, updated_at`
	workspaceColumnsOfW = `w.id, w.name, w.created_at, w.updated_at, m.role`
)

// WorkspaceRepository defines the data access contract for a workspace.
type WorkspaceRepository interface {
	Create(ctx context.Context, name string) (*model.Workspace, error)
	GetByID(ctx context.Context, id string) (*model.Workspace, error)
	ListOfUser(ctx context.Context, userID string) ([]model.Workspace, error)
	Rename(
		ctx context.Context,
		id string,
		name string,
		ifMatch *time.Time,
	) (*model.Workspace, error)
	Lock(ctx context.Context, id string) error
	ListAll(ctx context.Context) ([]model.InstanceWorkspace, error)
}

// Workspace is a SQL based implementation of WorkspaceRepository.
type Workspace struct {
	tx *sqlx.Tx
}

var _ WorkspaceRepository = (*Workspace)(nil)

// NewWorkspace creates a new Workspace repository instance.
func NewWorkspace(tx *sqlx.Tx) *Workspace {
	return &Workspace{tx: tx}
}

// Create writes one workspace.
func (r *Workspace) Create(
	ctx context.Context,
	name string,
) (*model.Workspace, error) {
	var entity model.Workspace

	query := `INSERT INTO public.workspaces (name) VALUES ($1) RETURNING ` + workspaceColumns + `;`

	if err := r.tx.GetContext(ctx, &entity, query, name); err != nil {
		return nil, fmt.Errorf("creating a Workspace: %w", err)
	}

	return &entity, nil
}

// GetByID retrieves the workspace with the given identifier.
func (r *Workspace) GetByID(ctx context.Context, id string) (*model.Workspace, error) {
	var entity model.Workspace

	query := `SELECT ` + workspaceColumns + ` FROM public.workspaces WHERE id = $1;`

	if err := r.tx.GetContext(ctx, &entity, query, id); err != nil {
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

	if err := r.tx.SelectContext(ctx, &entities, query, userID); err != nil {
		return nil, fmt.Errorf("listing the Workspaces of a User: %w", err)
	}

	return entities, nil
}

// Rename sets the name of the workspace. With a validator it changes the row only
// while the row still carries that moment, and answers precondition.ErrModified
// when the row moved since.
func (r *Workspace) Rename(
	ctx context.Context,
	id string,
	name string,
	ifMatch *time.Time,
) (*model.Workspace, error) {
	var entity model.Workspace

	query := `
		UPDATE public.workspaces SET name = $2
		WHERE id = $1 AND ($3::timestamptz IS NULL OR updated_at = $3)
		RETURNING ` + workspaceColumns + `;`

	err := r.tx.GetContext(ctx, &entity, query, id, name, ifMatch)
	if errors.Is(err, sql.ErrNoRows) {
		if ifMatch != nil {
			return nil, fmt.Errorf("renaming a Workspace: %w", precondition.ErrModified)
		}

		return nil, fmt.Errorf("renaming a Workspace: %w", errs.ErrWorkspaceNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("renaming a Workspace: %w", err)
	}

	return &entity, nil
}

// Lock holds the row of the workspace until the transaction ends, so two changes of
// its memberships run one after the other.
func (r *Workspace) Lock(ctx context.Context, id string) error {
	var held string

	err := r.tx.GetContext(
		ctx,
		&held,
		`SELECT id FROM public.workspaces WHERE id = $1 FOR UPDATE;`,
		id,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("locking a Workspace: %w", errs.ErrWorkspaceNotFound)
	}

	if err != nil {
		return fmt.Errorf("locking a Workspace: %w", err)
	}

	return nil
}

// ListAll retrieves every workspace of the instance with the count of its members and
// the plugins that it enables, ordered by the name, then by the identifier.
func (r *Workspace) ListAll(ctx context.Context) ([]model.InstanceWorkspace, error) {
	entities := []model.InstanceWorkspace{}

	query := `
		SELECT w.id, w.name, w.created_at, w.updated_at,
			(SELECT count(*) FROM public.workspace_members m WHERE m.workspace_id = w.id) AS member_count
		FROM public.workspaces w
		ORDER BY w.name, w.id;`

	if err := r.tx.SelectContext(ctx, &entities, query); err != nil {
		return nil, fmt.Errorf("listing every Workspace: %w", err)
	}

	enablements := []model.Enablement{}

	err := r.tx.SelectContext(ctx, &enablements, `
		SELECT workspace_id, plugin_id, created_at, updated_at FROM public.workspace_plugins
		ORDER BY workspace_id, plugin_id;`)
	if err != nil {
		return nil, fmt.Errorf("listing the plugins of every Workspace: %w", err)
	}

	enabled := make(map[string][]string, len(entities))
	for _, enablement := range enablements {
		enabled[enablement.WorkspaceID] = append(
			enabled[enablement.WorkspaceID],
			enablement.PluginID,
		)
	}

	for index := range entities {
		entities[index].PluginIDs = enabled[entities[index].ID]
		if entities[index].PluginIDs == nil {
			entities[index].PluginIDs = []string{}
		}
	}

	return entities, nil
}
