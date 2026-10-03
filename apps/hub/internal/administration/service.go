package administration

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

// UserChange holds the members of a merge patch of a user record. A nil member
// changes nothing.
type UserChange struct {
	Status        *model.Status
	Administrator *bool
}

// Service manages the user records of the instance.
type Service interface {
	ChangeUser(
		ctx context.Context,
		userID string,
		change UserChange,
		version *time.Time,
	) (*model.User, error)
}

// Manager is the implementation of Service over the repositories of the hub.
type Manager struct {
	db    *sqlx.DB
	users repository.UserRepository
}

var _ Service = (*Manager)(nil)

// NewManager creates a new Manager.
func NewManager(db *sqlx.DB, users repository.UserRepository) *Manager {
	return &Manager{db: db, users: users}
}

// ChangeUser writes the status and the mark of a user record. A change that leaves
// the instance with no active administrator answers errs.ErrAdministratorLast and
// changes nothing. A non-nil version refuses a record that moved after the client
// read it.
func (m *Manager) ChangeUser(
	ctx context.Context,
	userID string,
	change UserChange,
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
