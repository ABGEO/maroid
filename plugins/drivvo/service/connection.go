package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// VehicleLister lists the vehicles of the car log account with a token.
type VehicleLister interface {
	Vehicles(ctx context.Context, token string) ([]model.Vehicle, error)
}

// Connections connects the car log account of the acting workspace. It collects nothing:
// the schedule collects each hour.
type Connections struct {
	account *Account
	client  VehicleLister
}

// NewConnections creates a new Connections.
func NewConnections(account *Account, client VehicleLister) *Connections {
	return &Connections{account: account, client: client}
}

// Status reads the connection of the acting workspace. A connected workspace lists the
// vehicles at the car log service, which proves that the session still works. A session
// that no sign in renews reads as not connected.
func (c *Connections) Status(ctx context.Context) (*model.Connection, error) {
	credentialsStored, err := c.account.CredentialsStored(ctx)
	if err != nil {
		return nil, err
	}

	vehicles, err := withToken(ctx, c.account, func(token string) ([]model.Vehicle, error) {
		return c.client.Vehicles(ctx, token)
	})
	if errors.Is(err, errs.ErrNotConnected) || isRejection(err) {
		return &model.Connection{
			State:             model.ConnectionStateNotConnected,
			CredentialsStored: credentialsStored,
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("listing the vehicles: %w", err)
	}

	return connected(vehicles), nil
}

// Connect signs in with the stored credentials, replaces the stored session, and lists the
// vehicles with the new token. A refusal answers as the refusal it names, because the editor
// asked for a sign in.
func (c *Connections) Connect(ctx context.Context) (*model.Connection, error) {
	session, err := c.account.Connect(ctx)
	if err != nil {
		return nil, err
	}

	vehicles, err := c.client.Vehicles(ctx, session.Token)
	if errors.Is(err, errs.ErrSessionRejected) {
		return nil, fmt.Errorf(
			"%w: the service rejected a new session: %w",
			errs.ErrRequestFailed,
			err,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("listing the vehicles: %w", err)
	}

	return connected(vehicles), nil
}

// Disconnect removes the session of the acting workspace. The settings and the car log
// copy stay.
func (c *Connections) Disconnect(ctx context.Context) error {
	return c.account.Disconnect(ctx)
}

// isRejection reports a call whose session no sign in renews: the service refused the
// stored credentials, or the workspace stores none.
func isRejection(err error) bool {
	return IsRefusal(err) || errors.Is(err, errs.ErrCredentialsAbsent)
}

func connected(vehicles []model.Vehicle) *model.Connection {
	return &model.Connection{
		State:             model.ConnectionStateConnected,
		CredentialsStored: true,
		Vehicles:          vehicles,
	}
}
