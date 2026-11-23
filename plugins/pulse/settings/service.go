// Package settings turns pulse on and off for a person, and pauses it.
package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/plugins/pulse/catalog"
	"github.com/abgeo/maroid/plugins/pulse/errs"
	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/repository"
	"github.com/abgeo/maroid/plugins/pulse/round"
)

// Service reads and writes the settings of the acting user.
type Service struct {
	db     pluginapi.PluginDB
	rounds *round.Service
	now    func() time.Time
}

// NewService creates a Service. The rounds expire the open rounds at a turn-off.
func NewService(db pluginapi.PluginDB, rounds *round.Service, now func() time.Time) *Service {
	return &Service{db: db, rounds: rounds, now: now}
}

// Absent is the moment of the settings of a person who never wrote them. Its entity
// tag is "0".
func Absent() time.Time {
	return time.Unix(0, 0).UTC()
}

// Get answers the settings. A person with no row reads pulse off at the moment
// Absent, and the read writes nothing.
func (s *Service) Get(ctx context.Context) (*model.Settings, error) {
	var current *model.Settings

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var getErr error

		current, getErr = repository.NewSettings(tx).Get(ctx)
		if getErr != nil {
			return fmt.Errorf("reading the settings: %w", getErr)
		}

		return nil
	})
	if errors.Is(err, sql.ErrNoRows) {
		return &model.Settings{UpdatedAt: Absent()}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading the settings: %w", err)
	}

	return current, nil
}

// Put turns pulse on or off and sets the pause, under the validator. The first
// turn-on copies the defaults, and a turn-off expires every open round.
func (s *Service) Put(
	ctx context.Context,
	enabled bool,
	pausedUntil *time.Time,
	ifMatch *time.Time,
) (*model.Settings, error) {
	var (
		current *model.Settings
		closed  []round.Closed
	)

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		row, guard, txErr := s.lockOrCreate(ctx, tx, ifMatch)
		if txErr != nil {
			return txErr
		}

		if closed, txErr = s.apply(ctx, tx, row, enabled); txErr != nil {
			return txErr
		}

		row.PausedUntil = pausedUntil
		current = row

		return repository.NewSettings(tx).Update(ctx, row, guard)
	})
	if err != nil {
		return nil, fmt.Errorf("writing the settings: %w", err)
	}

	s.rounds.Expire(ctx, closed)

	return current, nil
}

// Pause stops every check-in until the moment.
func (s *Service) Pause(ctx context.Context, until time.Time) error {
	return s.setPause(ctx, &until)
}

// Resume ends the pause.
func (s *Service) Resume(ctx context.Context) error {
	return s.setPause(ctx, nil)
}

// lockOrCreate answers the row of the person and the validator that guards its
// write. With no row, only the validator "0" or none creates it, and the new row
// needs no further guard.
func (s *Service) lockOrCreate(
	ctx context.Context,
	tx *sqlx.Tx,
	ifMatch *time.Time,
) (*model.Settings, *time.Time, error) {
	repo := repository.NewSettings(tx)

	row, err := repo.Lock(ctx)
	if err == nil {
		return row, ifMatch, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, fmt.Errorf("locking the settings: %w", err)
	}

	// The validator "0" stands for the row that does not exist yet.
	if ifMatch != nil && !ifMatch.Equal(Absent()) {
		return nil, nil, precondition.ErrModified
	}

	row, err = repo.Insert(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("creating the settings: %w", err)
	}

	return row, nil, nil
}

// apply runs the effects of a change of the flag on the row.
func (s *Service) apply(
	ctx context.Context,
	tx *sqlx.Tx,
	row *model.Settings,
	enabled bool,
) ([]round.Closed, error) {
	switch {
	case enabled && !row.Enabled:
		row.Enabled = true

		return nil, s.turnOn(ctx, tx, row)
	case !enabled && row.Enabled:
		row.Enabled = false

		closed, err := round.CloseOpen(ctx, tx, nil)
		if err != nil {
			return nil, fmt.Errorf("closing the open rounds: %w", err)
		}

		return closed, nil
	default:
		return nil, nil
	}
}

func (s *Service) turnOn(ctx context.Context, tx *sqlx.Tx, row *model.Settings) error {
	now := s.now()

	if row.SeededAt == nil {
		if err := catalog.Seed(ctx, tx, now); err != nil {
			return fmt.Errorf("copying the defaults: %w", err)
		}

		row.SeededAt = &now
	}

	// A turned-on check-in fires first at its next occurrence.
	if err := repository.NewCheckin(tx).SetEveryFiresAfter(ctx, now); err != nil {
		return fmt.Errorf("moving the marks of the check-ins: %w", err)
	}

	return nil
}

func (s *Service) setPause(ctx context.Context, until *time.Time) error {
	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		repo := repository.NewSettings(tx)

		row, err := repo.Lock(ctx)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && !row.Enabled) {
			return errs.ErrPulseOff
		}

		if err != nil {
			return fmt.Errorf("locking the settings: %w", err)
		}

		row.PausedUntil = until

		return repo.Update(ctx, row, nil)
	})
	if err != nil {
		return fmt.Errorf("writing the pause: %w", err)
	}

	return nil
}
