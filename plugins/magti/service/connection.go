package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
)

// AssignmentStore holds the assigned subscriber of the acting workspace.
type AssignmentStore interface {
	Assignment(ctx context.Context) (model.Assignment, error)
	Assign(
		ctx context.Context,
		number string,
		kind model.SubscriberKind,
		ifMatch *time.Time,
	) (model.Assignment, error)
	Unassign(ctx context.Context) error
}

// Connections connects the telecom account of the acting workspace, and assigns one of
// its subscribers to the workspace. It collects nothing: an administrator collects the
// history from the command line, and the schedule collects each day.
type Connections struct {
	account *Account
	client  APIClientService
	store   AssignmentStore
}

// NewConnections creates a new Connections instance.
func NewConnections(account *Account, client APIClientService, store AssignmentStore) *Connections {
	return &Connections{account: account, client: client, store: store}
}

// Status reads the connection of the acting workspace. A connected workspace reads the
// subscribers of the account, which also proves that the session still works. A session
// that no sign in can renew reads as not connected.
func (c *Connections) Status(ctx context.Context) (*model.Connection, error) {
	credentialsStored, err := c.credentialsStored(ctx)
	if err != nil {
		return nil, err
	}

	subscribers, err := callLocked(
		ctx,
		c.account,
		func(session model.Session) ([]model.Subscriber, error) {
			return c.client.Subscribers(ctx, session)
		},
	)
	if isRejection(err) || errors.Is(err, errs.ErrNotConnected) {
		return &model.Connection{
			State:             model.ConnectionStateNotConnected,
			CredentialsStored: credentialsStored,
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("listing the subscribers: %w", err)
	}

	return c.connection(ctx, subscribers, credentialsStored)
}

// Connect signs in with the stored credentials and stores the session. The new session
// replaces the old one, and the assigned subscriber stays. A refusal answers as the
// refusal it names, because the editor asked for a sign in.
func (c *Connections) Connect(ctx context.Context) (*model.Connection, error) {
	unlock, err := c.account.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	session, err := c.account.signIn(ctx)
	if err != nil {
		return nil, err
	}

	subscribers, err := c.client.Subscribers(ctx, session)
	if errors.Is(err, errs.ErrSessionRejected) {
		return nil, fmt.Errorf("%w: the portal rejected a new session", errs.ErrRequestFailed)
	}

	if err != nil {
		return nil, fmt.Errorf("listing the subscribers: %w", err)
	}

	return c.connection(ctx, subscribers, true)
}

// Disconnect removes the session and the assignment of the acting workspace. The
// settings, the traffic, and the money entries stay.
func (c *Connections) Disconnect(ctx context.Context) error {
	unlock, err := c.account.lock(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	if err = c.account.forgetSession(ctx); err != nil {
		return err
	}

	if err := c.store.Unassign(ctx); err != nil {
		return fmt.Errorf("removing the assignment: %w", err)
	}

	return nil
}

// Assign assigns a mobile subscriber or a home line of the account to the acting
// workspace, and collects nothing. A non-nil ifMatch guards the write with the moment of
// the assignment that the client read.
func (c *Connections) Assign(
	ctx context.Context,
	number string,
	ifMatch *time.Time,
) (*model.Connection, error) {
	subscribers, err := callLocked(
		ctx,
		c.account,
		func(session model.Session) ([]model.Subscriber, error) {
			return c.client.Subscribers(ctx, session)
		},
	)
	if err = asNotConnected(err); err != nil {
		return nil, fmt.Errorf("listing the subscribers: %w", err)
	}

	credentialsStored, err := c.credentialsStored(ctx)
	if err != nil {
		return nil, err
	}

	subscriber, found := findSubscriber(subscribers, number)

	switch {
	case !found:
		return nil, errs.ErrSubscriberUnknown
	case !subscriber.Kind.Assignable():
		return nil, errs.ErrSubscriberUnsupported
	}

	assignment, err := c.store.Assign(ctx, number, subscriber.Kind, ifMatch)
	if err != nil {
		return nil, fmt.Errorf("storing the assignment: %w", err)
	}

	return connectionOf(subscribers, &assignment, credentialsStored), nil
}

func (c *Connections) connection(
	ctx context.Context,
	subscribers []model.Subscriber,
	credentialsStored bool,
) (*model.Connection, error) {
	assignment, err := c.store.Assignment(ctx)

	switch {
	case errors.Is(err, errs.ErrNotAssigned):
		return connectionOf(subscribers, nil, credentialsStored), nil
	case err != nil:
		return nil, fmt.Errorf("reading the assignment: %w", err)
	}

	return connectionOf(subscribers, &assignment, credentialsStored), nil
}

func (c *Connections) credentialsStored(ctx context.Context) (bool, error) {
	_, err := c.account.credentials(ctx)

	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, errs.ErrCredentialsAbsent):
		return false, nil
	default:
		return false, err
	}
}

// asNotConnected answers a session that no sign in renews as errs.ErrNotConnected, and
// any other error as it is. The refusal stays out, because only a connect answers it.
func asNotConnected(err error) error {
	if isRejection(err) {
		return fmt.Errorf("%w: the sign in was refused", errs.ErrNotConnected)
	}

	return err
}

// isRejection reports a call whose session no sign in renews: the portal refused the
// stored credentials, or the workspace stores none.
func isRejection(err error) bool {
	return isRefusal(err) || errors.Is(err, errs.ErrCredentialsAbsent)
}

func findSubscriber(subscribers []model.Subscriber, number string) (model.Subscriber, bool) {
	for _, subscriber := range subscribers {
		if subscriber.Number == number {
			return subscriber, true
		}
	}

	return model.Subscriber{}, false
}

func connectionOf(
	subscribers []model.Subscriber,
	assignment *model.Assignment,
	credentialsStored bool,
) *model.Connection {
	connection := &model.Connection{
		State:             model.ConnectionStateConnected,
		CredentialsStored: credentialsStored,
		Subscribers:       make([]model.Subscriber, 0, len(subscribers)),
		Assignment:        assignment,
	}

	for _, subscriber := range subscribers {
		subscriber.Assigned = assignment != nil && subscriber.Number == assignment.SubscriberNumber
		connection.Subscribers = append(connection.Subscribers, subscriber)
	}

	return connection
}
