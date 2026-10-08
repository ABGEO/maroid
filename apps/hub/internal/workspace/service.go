package workspace

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// Service creates, reads, and changes a workspace and its members. Each method reads
// the acting user and the acting workspace from the context.
type Service interface {
	Create(ctx context.Context, name string) (*model.Workspace, error)
	ListOfUser(ctx context.Context) ([]model.Workspace, error)
	Get(ctx context.Context) (*model.Workspace, error)
	Rename(ctx context.Context, name string, version *time.Time) (*model.Workspace, error)
	Members(ctx context.Context) ([]model.Member, error)
	Member(ctx context.Context, userID string) (*model.Member, error)
	Candidates(ctx context.Context) ([]model.User, error)
	AddMember(ctx context.Context, userID string, role pluginapi.Role) (*model.Member, error)
	RemoveMember(ctx context.Context, userID string) error
	ChangeRole(
		ctx context.Context,
		userID string,
		role pluginapi.Role,
		version *time.Time,
	) (*model.Member, error)
}

// Catalog lists every workspace of the instance, for an administrator. Manager
// satisfies it beside Service, because it reads no acting workspace.
type Catalog interface {
	ListAll(ctx context.Context) ([]model.InstanceWorkspace, error)
}

// Manager is the implementation of Service over the repositories of the hub.
type Manager struct {
	db *sqlx.DB
}

var (
	_ Service = (*Manager)(nil)
	_ Catalog = (*Manager)(nil)
)

// NewManager creates a new Manager.
func NewManager(db *sqlx.DB) *Manager {
	return &Manager{db: db}
}

// Create writes the workspace and the membership of the acting user in one
// transaction, so no workspace exists without a member.
func (m *Manager) Create(ctx context.Context, name string) (*model.Workspace, error) {
	var created *model.Workspace

	err := database.WithTx(ctx, m.db, func(tx *sqlx.Tx) error {
		workspace, err := repository.NewWorkspace(tx).Create(ctx, name)
		if err != nil {
			return fmt.Errorf("creating the workspace: %w", err)
		}

		if _, err = repository.NewWorkspaceMember(tx).Add(
			ctx,
			workspace.ID,
			pluginapi.ActingUserFromContext(ctx),
			pluginapi.RoleManager,
		); err != nil {
			return fmt.Errorf("adding the first member: %w", err)
		}

		created = workspace

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("creating a workspace: %w", err)
	}

	return created, nil
}

// ListOfUser lists the workspaces of the acting user.
func (m *Manager) ListOfUser(ctx context.Context) ([]model.Workspace, error) {
	workspaces, err := database.FetchTx(ctx, m.db, func(tx *sqlx.Tx) ([]model.Workspace, error) {
		return repository.NewWorkspace(tx).ListOfUser(ctx, pluginapi.ActingUserFromContext(ctx))
	})
	if err != nil {
		return nil, fmt.Errorf("listing the workspaces of the acting user: %w", err)
	}

	return workspaces, nil
}

// Get reads the acting workspace.
func (m *Manager) Get(ctx context.Context) (*model.Workspace, error) {
	workspace, err := database.FetchTx(ctx, m.db, func(tx *sqlx.Tx) (*model.Workspace, error) {
		return repository.NewWorkspace(tx).GetByID(ctx, pluginapi.ActingWorkspaceFromContext(ctx))
	})
	if err != nil {
		return nil, fmt.Errorf("reading the acting workspace: %w", err)
	}

	return workspace, nil
}

// Rename sets the name of the acting workspace.
func (m *Manager) Rename(
	ctx context.Context,
	name string,
	version *time.Time,
) (*model.Workspace, error) {
	workspace, err := database.FetchTx(ctx, m.db, func(tx *sqlx.Tx) (*model.Workspace, error) {
		return repository.NewWorkspace(tx).Rename(
			ctx,
			pluginapi.ActingWorkspaceFromContext(ctx),
			name,
			version,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("renaming the acting workspace: %w", err)
	}

	return workspace, nil
}

// ListAll lists every workspace of the instance, each with the count of its members
// and the plugins that it enables. An administrator reads it, and the caller checks
// the mark.
func (m *Manager) ListAll(ctx context.Context) ([]model.InstanceWorkspace, error) {
	workspaces, err := database.FetchTx(
		ctx,
		m.db,
		func(tx *sqlx.Tx) ([]model.InstanceWorkspace, error) {
			return repository.NewWorkspace(tx).ListAll(ctx)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("listing the workspaces of the instance: %w", err)
	}

	return workspaces, nil
}

// Members lists the members of the acting workspace.
func (m *Manager) Members(ctx context.Context) ([]model.Member, error) {
	members, err := database.FetchTx(ctx, m.db, func(tx *sqlx.Tx) ([]model.Member, error) {
		return repository.NewWorkspaceMember(tx).
			List(ctx, pluginapi.ActingWorkspaceFromContext(ctx))
	})
	if err != nil {
		return nil, fmt.Errorf("listing the members of the acting workspace: %w", err)
	}

	return members, nil
}

// Member reads one membership of the acting workspace.
func (m *Manager) Member(ctx context.Context, userID string) (*model.Member, error) {
	member, err := database.FetchTx(ctx, m.db, func(tx *sqlx.Tx) (*model.Member, error) {
		return repository.NewWorkspaceMember(tx).Get(
			ctx, pluginapi.ActingWorkspaceFromContext(ctx), userID,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("reading a member of the acting workspace: %w", err)
	}

	return member, nil
}

// Candidates lists every active user record that is no member of the acting
// workspace.
func (m *Manager) Candidates(ctx context.Context) ([]model.User, error) {
	candidates, err := database.FetchTx(ctx, m.db, func(tx *sqlx.Tx) ([]model.User, error) {
		return repository.NewWorkspaceMember(tx).Candidates(
			ctx, pluginapi.ActingWorkspaceFromContext(ctx),
		)
	})
	if err != nil {
		return nil, fmt.Errorf("listing the candidates of the acting workspace: %w", err)
	}

	return candidates, nil
}

// AddMember adds an active user record to the acting workspace with the role. A
// record that is absent or blocked answers errs.ErrUserNotFound.
func (m *Manager) AddMember(
	ctx context.Context,
	userID string,
	role pluginapi.Role,
) (*model.Member, error) {
	added, err := database.FetchTx(ctx, m.db, func(tx *sqlx.Tx) (*model.Member, error) {
		if _, err := repository.NewUser(tx).GetActiveByID(ctx, userID); err != nil {
			return nil, fmt.Errorf("reading the record to add: %w", err)
		}

		member, err := repository.NewWorkspaceMember(tx).Add(
			ctx,
			pluginapi.ActingWorkspaceFromContext(ctx),
			userID,
			role,
		)
		if err != nil {
			return nil, fmt.Errorf("adding the member: %w", err)
		}

		return member, nil
	})
	if err != nil {
		return nil, fmt.Errorf("adding a member to the acting workspace: %w", err)
	}

	return added, nil
}

// RemoveMember takes one membership out of the acting workspace. When userID names
// the acting user, it is a leave. A removal that leaves no manager answers
// errs.ErrManagerLast and removes nothing.
func (m *Manager) RemoveMember(ctx context.Context, userID string) error {
	err := m.keepingAManager(ctx, func(tx *sqlx.Tx, workspaceID string) error {
		return repository.NewWorkspaceMember(tx).Remove(ctx, workspaceID, userID)
	})
	if err != nil {
		return fmt.Errorf("removing a member from the acting workspace: %w", err)
	}

	return nil
}

// ChangeRole sets the role of a member of the acting workspace. A change that leaves
// no manager answers errs.ErrManagerLast and changes nothing.
func (m *Manager) ChangeRole(
	ctx context.Context,
	userID string,
	role pluginapi.Role,
	version *time.Time,
) (*model.Member, error) {
	var changed *model.Member

	err := m.keepingAManager(ctx, func(tx *sqlx.Tx, workspaceID string) error {
		var err error

		changed, err = repository.NewWorkspaceMember(tx).
			ChangeRole(ctx, workspaceID, userID, role, version)

		return err //nolint:wrapcheck // keepingAManager wraps it.
	})
	if err != nil {
		return nil, fmt.Errorf("changing the role of a member of the acting workspace: %w", err)
	}

	return changed, nil
}

// keepingAManager runs one change of the memberships of the acting workspace after a
// lock of the workspace, and rolls it back when it leaves no manager. Without the
// lock, two managers who demote each other at the same moment each count two
// managers, and both commits leave none.
func (m *Manager) keepingAManager(
	ctx context.Context,
	change func(tx *sqlx.Tx, workspaceID string) error,
) error {
	workspaceID := pluginapi.ActingWorkspaceFromContext(ctx)

	err := database.WithTx(ctx, m.db, func(tx *sqlx.Tx) error {
		if err := repository.NewWorkspace(tx).Lock(ctx, workspaceID); err != nil {
			return fmt.Errorf("locking the workspace: %w", err)
		}

		if err := change(tx, workspaceID); err != nil {
			return err
		}

		managers, err := repository.NewWorkspaceMember(tx).CountManagers(ctx, workspaceID)
		if err != nil {
			return fmt.Errorf("counting the managers: %w", err)
		}

		if managers == 0 {
			return errs.ErrManagerLast
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("changing the memberships: %w", err)
	}

	return nil
}
