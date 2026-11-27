package service

import (
	"context"
	"errors"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/errs"
	"github.com/abgeo/maroid/plugins/pensions/model"
)

// Connections connects the pension account of the acting user. It collects nothing: the
// schedule collects each month, and an administrator collects the history from the
// command line.
type Connections struct {
	account *account
	client  APIClientService
}

// NewConnections creates a new Connections instance.
func NewConnections(
	client APIClientService,
	settings pluginapi.PluginSettings,
	secrets pluginapi.PluginSecrets,
) *Connections {
	return &Connections{
		account: &account{client: client, settings: settings, secrets: secrets},
		client:  client,
	}
}

// Status reads the connection of the acting user. A connected user reads the
// participant, which also proves that the session still works. A session that no sign in
// can renew reads as not connected.
func (c *Connections) Status(ctx context.Context) (*model.Connection, error) {
	credentialsStored, err := c.credentialsStored(ctx)
	if err != nil {
		return nil, err
	}

	notConnected := &model.Connection{
		State:             model.ConnectionStateNotConnected,
		CredentialsStored: credentialsStored,
	}

	session, err := c.account.session(ctx)
	if errors.Is(err, errs.ErrNotConnected) {
		return notConnected, nil
	}

	if err != nil {
		return nil, err
	}

	participant, _, err := callWith(ctx, c.account, session,
		func(session model.Session) (model.Participant, error) {
			return c.client.Participant(ctx, session)
		})
	if isRejection(err) {
		return notConnected, nil
	}

	if err != nil {
		return nil, err
	}

	return connectionOf(participant, credentialsStored), nil
}

// Connect signs in with the stored credentials and stores the session. The new session
// replaces the old one. A rejection answers as the credentials it names, because the
// person asked for a sign in.
func (c *Connections) Connect(ctx context.Context) (*model.Connection, error) {
	session, err := c.account.signIn(ctx)
	if err != nil {
		return nil, err
	}

	participant, _, err := callWith(ctx, c.account, session,
		func(session model.Session) (model.Participant, error) {
			return c.client.Participant(ctx, session)
		})
	if err != nil {
		return nil, err
	}

	return connectionOf(participant, true), nil
}

// Disconnect removes the session of the acting user. The settings and the collected
// data stay.
func (c *Connections) Disconnect(ctx context.Context) error {
	return c.account.forgetSession(ctx)
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

// isRejection reports a call whose session no sign in renews: the pension service
// rejected the stored credentials, or the user stores none. callWith removed the session
// in both cases.
func isRejection(err error) bool {
	return errors.Is(err, errs.ErrCredentialsRejected) || errors.Is(err, errs.ErrCredentialsAbsent)
}

func connectionOf(participant model.Participant, credentialsStored bool) *model.Connection {
	return &model.Connection{
		State:             model.ConnectionStateConnected,
		CredentialsStored: credentialsStored,
		Holder: &model.Holder{
			FullName:   participant.FullName,
			AssetValue: participant.Summary.AssetValue,
		},
	}
}
