package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/abgeo/maroid/plugins/magti/config"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
)

// AssignmentReader reads the assigned subscriber of the acting workspace.
type AssignmentReader interface {
	Assignment(ctx context.Context) (model.Assignment, error)
}

// States reads the subscriber state of the assigned subscriber from the portal at each
// read. Maroid keeps no copy of it.
type States struct {
	config  *config.Config
	account *Account
	client  APIClientService
	store   AssignmentReader
	now     func() time.Time
}

// NewStates creates a new States. now gives the moment of a read.
func NewStates(
	cfg *config.Config,
	account *Account,
	client APIClientService,
	store AssignmentReader,
	now func() time.Time,
) *States {
	return &States{config: cfg, account: account, client: client, store: store, now: now}
}

// Read answers the state of the assigned subscriber of the acting workspace. It answers
// errs.ErrNotAssigned with no assignment and errs.ErrNotConnected with no session. A read
// that passes state_timeout answers errs.ErrRequestFailed, so it ends before the hub ends
// the widget.
func (s *States) Read(ctx context.Context) (model.State, error) {
	assignment, err := s.store.Assignment(ctx)
	if err != nil {
		return model.State{}, fmt.Errorf("reading the assigned subscriber: %w", err)
	}

	bounded, cancel := context.WithTimeout(ctx, s.config.StateTimeout)
	defer cancel()

	state, err := callLocked(bounded, s.account, func(session model.Session) (model.State, error) {
		return s.read(bounded, session, assignment)
	})

	if errors.Is(bounded.Err(), context.DeadlineExceeded) {
		return model.State{}, fmt.Errorf(
			"%w: reading the state: %w",
			errs.ErrRequestFailed,
			context.DeadlineExceeded,
		)
	}

	if err = asNotConnected(err); err != nil {
		return model.State{}, err
	}

	if state.Plan != nil && state.Plan.RenewsAt != nil {
		state.Plan.DaysLeft = model.DaysUntil(s.now(), *state.Plan.RenewsAt)
	}

	return state, nil
}

func (s *States) read(
	ctx context.Context,
	session model.Session,
	assignment model.Assignment,
) (model.State, error) {
	number := assignment.SubscriberNumber

	if _, err := s.client.Select(ctx, session, number); err != nil {
		return model.State{}, fmt.Errorf("selecting the subscriber: %w", err)
	}

	read := s.client.MobileState
	if assignment.IsHomeLine() {
		read = s.client.HomeLineState
	}

	state, err := read(ctx, session, number)
	if err != nil {
		return model.State{}, fmt.Errorf("reading the state: %w", err)
	}

	return state, nil
}
