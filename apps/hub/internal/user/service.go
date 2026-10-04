package user

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// Change holds the members of a merge patch of a user record. A nil member changes
// nothing.
type Change struct {
	FirstName     *string
	LastName      *string
	Status        *model.Status
	Administrator *bool
}

// Invitation is the token that redeems an invitation, and the moment it expires. The
// token appears here one time and never again.
type Invitation struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
}

// Service manages the user records of the instance and the allowlist of each.
type Service interface {
	List(ctx context.Context) ([]model.User, error)
	Get(ctx context.Context, userID string) (*model.User, error)
	Create(ctx context.Context, request auth.InviteRequest) (*model.User, *Invitation, error)
	Invite(ctx context.Context, userID string) (*Invitation, error)
	Change(
		ctx context.Context,
		userID string,
		change Change,
		version *time.Time,
	) (*model.User, error)
	AllowedPlugins(ctx context.Context, userID string) ([]model.AllowedPlugin, error)
	AllowPlugin(
		ctx context.Context,
		userID string,
		pluginID string,
	) (*model.AllowedPlugin, bool, error)
	DisallowPlugin(ctx context.Context, userID string, pluginID string) error
}

// Inviter issues an invitation, and writes the user record when the request names none.
// auth.Service satisfies it.
type Inviter interface {
	Invite(
		ctx context.Context,
		request auth.InviteRequest,
		ttl time.Duration,
	) (*auth.InviteResult, error)
}

// PluginCatalog lists the plugins that the hub loaded. registry.PluginRegistry
// satisfies it.
type PluginCatalog interface {
	All() []pluginapi.Plugin
}

// Repositories holds the data access that the service reads and writes.
type Repositories struct {
	Users   repository.UserRepository
	Allowed repository.AllowedPluginRepository
}

// Manager is the implementation of Service over the repositories of the hub.
type Manager struct {
	db            *sqlx.DB
	users         repository.UserRepository
	allowed       repository.AllowedPluginRepository
	inviter       Inviter
	plugins       PluginCatalog
	invitationTTL time.Duration
}

var _ Service = (*Manager)(nil)

// NewManager creates a new Manager.
func NewManager(
	db *sqlx.DB,
	repositories Repositories,
	inviter Inviter,
	plugins PluginCatalog,
	invitationTTL time.Duration,
) *Manager {
	return &Manager{
		db:            db,
		users:         repositories.Users,
		allowed:       repositories.Allowed,
		inviter:       inviter,
		plugins:       plugins,
		invitationTTL: invitationTTL,
	}
}

// List lists every user record of the instance.
func (m *Manager) List(ctx context.Context) ([]model.User, error) {
	users, err := m.users.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing the user records: %w", err)
	}

	return users, nil
}

// Get reads one user record, active or blocked.
func (m *Manager) Get(ctx context.Context, userID string) (*model.User, error) {
	user, err := m.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("reading a user record: %w", err)
	}

	return user, nil
}

// Create writes a user record, its first workspace, and its invitation, the way
// maroid user invite does.
func (m *Manager) Create(
	ctx context.Context,
	request auth.InviteRequest,
) (*model.User, *Invitation, error) {
	request.UserID = ""

	if request.Administrator && len(request.AllowedPlugins) > 0 {
		return nil, nil, errs.ErrAdministratorAllowlist
	}

	for index, pluginID := range request.AllowedPlugins {
		if !m.loaded(pluginID) {
			return nil, nil, &NotLoadedError{Index: index, PluginID: pluginID}
		}
	}

	invitation, err := m.issue(ctx, request)
	if err != nil {
		return nil, nil, err
	}

	user, err := m.Get(ctx, invitation.UserID)
	if err != nil {
		return nil, nil, err
	}

	return user, invitation, nil
}

// Invite issues a new invitation for a user record that exists. An invitation issued
// before stays valid.
func (m *Manager) Invite(ctx context.Context, userID string) (*Invitation, error) {
	if _, err := m.Get(ctx, userID); err != nil {
		return nil, err
	}

	return m.issue(ctx, auth.InviteRequest{UserID: userID})
}

// AllowedPlugins lists the allowlist of a user record.
func (m *Manager) AllowedPlugins(
	ctx context.Context,
	userID string,
) ([]model.AllowedPlugin, error) {
	if _, err := m.Get(ctx, userID); err != nil {
		return nil, err
	}

	allowed, err := m.allowed.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing the allowlist: %w", err)
	}

	return allowed, nil
}

// AllowPlugin puts a loaded plugin on the allowlist of a user record. The second
// answer is true when it wrote the row.
func (m *Manager) AllowPlugin(
	ctx context.Context,
	userID string,
	pluginID string,
) (*model.AllowedPlugin, bool, error) {
	if err := m.holdsAllowlist(ctx, userID); err != nil {
		return nil, false, err
	}

	if !m.loaded(pluginID) {
		return nil, false, fmt.Errorf("%w: %s", errs.ErrPluginNotLoaded, pluginID)
	}

	allowed, created, err := m.allowed.Add(ctx, userID, pluginID)
	if err != nil {
		return nil, false, fmt.Errorf("adding to the allowlist: %w", err)
	}

	return allowed, created, nil
}

// DisallowPlugin takes a plugin off the allowlist of a user record. No enablement
// changes, because no table holds which manager enabled a plugin.
func (m *Manager) DisallowPlugin(ctx context.Context, userID string, pluginID string) error {
	if err := m.holdsAllowlist(ctx, userID); err != nil {
		return err
	}

	if err := m.allowed.Remove(ctx, userID, pluginID); err != nil {
		return fmt.Errorf("removing from the allowlist: %w", err)
	}

	return nil
}

// Change writes the status and the mark of a user record. A change that leaves
// the instance with no active administrator answers errs.ErrAdministratorLast and
// changes nothing. A non-nil version refuses a record that moved after the client
// read it.
func (m *Manager) Change(
	ctx context.Context,
	userID string,
	change Change,
	version *time.Time,
) (*model.User, error) {
	var changed *model.User

	err := database.WithTx(ctx, m.db, func(tx *sqlx.Tx) error {
		// Two administrators who unmark each other at the same moment each count two
		// without the lock, and both commits leave none.
		if err := m.users.LockAdministrators(ctx, tx); err != nil {
			return fmt.Errorf("locking the administrators: %w", err)
		}

		var err error

		changed, err = m.users.Change(ctx, tx, userID, repository.UserChange{
			FirstName:     change.FirstName,
			LastName:      change.LastName,
			Status:        change.Status,
			Administrator: change.Administrator,
			IfMatch:       version,
		})
		if err != nil {
			return fmt.Errorf("changing the record: %w", err)
		}

		administrators, err := m.users.CountActiveAdministrators(ctx, tx)
		if err != nil {
			return fmt.Errorf("counting the administrators: %w", err)
		}

		if administrators == 0 {
			return errs.ErrAdministratorLast
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("changing a user record: %w", err)
	}

	return changed, nil
}

func (m *Manager) issue(ctx context.Context, request auth.InviteRequest) (*Invitation, error) {
	result, err := m.inviter.Invite(ctx, request, m.invitationTTL)
	if err != nil {
		return nil, fmt.Errorf("issuing the invitation: %w", err)
	}

	return &Invitation{
		UserID:    result.UserID,
		Token:     result.Token,
		ExpiresAt: time.Now().Add(m.invitationTTL),
	}, nil
}

// holdsAllowlist answers ErrUserNotFound for a record that does not exist, and
// ErrAdministratorAllowlist for an administrator, who turns on every plugin.
func (m *Manager) holdsAllowlist(ctx context.Context, userID string) error {
	record, err := m.Get(ctx, userID)
	if err != nil {
		return err
	}

	if record.IsAdministrator {
		return errs.ErrAdministratorAllowlist
	}

	return nil
}

func (m *Manager) loaded(pluginID string) bool {
	for _, plugin := range m.plugins.All() {
		if plugin.Meta().ID.String() == pluginID {
			return true
		}
	}

	return false
}

// NotLoadedError reports a plugin of the allowlist of a new record that the hub did not
// load, with its place in the list.
type NotLoadedError struct {
	Index    int
	PluginID string
}

func (e *NotLoadedError) Error() string {
	return fmt.Sprintf("%s: %s at %d", errs.ErrPluginNotLoaded, e.PluginID, e.Index)
}

// Unwrap answers ErrPluginNotLoaded, so a caller that checks the sentinel finds it.
func (e *NotLoadedError) Unwrap() error {
	return errs.ErrPluginNotLoaded
}
