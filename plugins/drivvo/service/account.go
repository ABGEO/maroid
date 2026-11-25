package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/drivvo/config"
	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// SecretSession names the workspace plugin secret that holds the session of the car log
// service.
const SecretSession = "session"

// SignInClient signs in to the car log service.
type SignInClient interface {
	SignIn(ctx context.Context, email, password string) (model.Session, error)
}

// Account reads the credentials and the session of the acting workspace, and signs in. The
// connections and the collector share one Account, so one sign in serves both.
type Account struct {
	client   SignInClient
	settings pluginapi.PluginSettings
	secrets  pluginapi.PluginSecrets
	now      func() time.Time

	mu    sync.Mutex
	locks map[string]chan struct{}
}

// NewAccount creates a new Account.
func NewAccount(
	client SignInClient,
	settings pluginapi.PluginSettings,
	secrets pluginapi.PluginSecrets,
	now func() time.Time,
) *Account {
	return &Account{
		client:   client,
		settings: settings,
		secrets:  secrets,
		now:      now,
		locks:    map[string]chan struct{}{},
	}
}

// Connect signs in with the stored credentials, and replaces the stored session.
func (a *Account) Connect(ctx context.Context) (model.Session, error) {
	unlock, err := a.lock(ctx)
	if err != nil {
		return model.Session{}, err
	}
	defer unlock()

	return a.signIn(ctx)
}

// Disconnect removes the stored session. It waits for a sign in that runs, so no session
// stays after both. The car log copy stays.
func (a *Account) Disconnect(ctx context.Context) error {
	unlock, err := a.lock(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	return a.forgetSession(ctx)
}

// Connected reports whether the acting workspace holds a session.
func (a *Account) Connected(ctx context.Context) (bool, error) {
	_, err := a.session(ctx)
	if errors.Is(err, errs.ErrNotConnected) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

// CredentialsStored reports whether the settings of the acting workspace hold an email
// address and a password.
func (a *Account) CredentialsStored(ctx context.Context) (bool, error) {
	_, err := a.credentials(ctx)
	if errors.Is(err, errs.ErrCredentialsAbsent) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

// withToken runs the call with the token of the acting workspace. A token past its end
// gives way to a sign in before the call. A rejected token gives way to one sign in, and
// the call runs again from its start. It answers errs.ErrNotConnected with no session.
func withToken[T any](
	ctx context.Context,
	acct *Account,
	run func(token string) (T, error),
) (T, error) {
	var none T

	session, err := acct.session(ctx)
	if err != nil {
		return none, err
	}

	if session.Expired(acct.now()) {
		if session, err = acct.renew(ctx, session.Token); err != nil {
			return none, fmt.Errorf("replacing an ended session: %w", err)
		}
	}

	answer, err := run(session.Token)
	if !errors.Is(err, errs.ErrSessionRejected) {
		return answer, err
	}

	if session, err = acct.renew(ctx, session.Token); err != nil {
		return none, fmt.Errorf("replacing a rejected session: %w", err)
	}

	answer, err = run(session.Token)
	if errors.Is(err, errs.ErrSessionRejected) {
		return none, fmt.Errorf(
			"%w: the service rejected a new session: %w",
			errs.ErrRequestFailed,
			err,
		)
	}

	return answer, err
}

// renew signs in under the lock of the acting workspace, unless a call that held the lock
// before stored a token that differs from the stale one and has not ended.
func (a *Account) renew(ctx context.Context, stale string) (model.Session, error) {
	unlock, err := a.lock(ctx)
	if err != nil {
		return model.Session{}, err
	}
	defer unlock()

	current, err := a.session(ctx)
	if err != nil {
		return model.Session{}, err
	}

	if current.Token != stale && !current.Expired(a.now()) {
		return current, nil
	}

	return a.signIn(ctx)
}

// lock waits until no other sign in of the acting workspace runs, or until the context
// ends. The answered function releases it.
func (a *Account) lock(ctx context.Context) (func(), error) {
	workspaceID := pluginapi.ActingWorkspaceFromContext(ctx)

	a.mu.Lock()

	held, found := a.locks[workspaceID]
	if !found {
		held = make(chan struct{}, 1)
		a.locks[workspaceID] = held
	}

	a.mu.Unlock()

	select {
	case held <- struct{}{}:
		return func() { <-held }, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: waiting for the sign in: %w", errs.ErrRequestFailed, ctx.Err())
	}
}

func (a *Account) credentials(ctx context.Context) (*config.WorkspaceSettings, error) {
	values, err := a.settings.Get(ctx)
	if errors.Is(err, pluginapi.ErrSettingsAbsent) {
		return nil, errs.ErrCredentialsAbsent
	}

	if err != nil {
		return nil, fmt.Errorf("reading the settings of the workspace: %w", err)
	}

	credentials := new(config.WorkspaceSettings)
	if err = pluginconfig.DecodeAndValidateSettings(values, credentials); err != nil {
		return nil, fmt.Errorf("decoding the settings of the workspace: %w", err)
	}

	if credentials.Email == "" || credentials.Password == "" {
		return nil, errs.ErrCredentialsAbsent
	}

	return credentials, nil
}

// session answers the stored session, or errs.ErrNotConnected when the workspace holds none.
func (a *Account) session(ctx context.Context) (model.Session, error) {
	stored, err := a.secrets.Get(ctx, pluginapi.SecretScopeWorkspace, SecretSession)
	if errors.Is(err, pluginapi.ErrSecretAbsent) {
		return model.Session{}, errs.ErrNotConnected
	}

	if err != nil {
		return model.Session{}, fmt.Errorf("reading the session: %w", err)
	}

	session, err := model.ParseSession(stored)
	if err != nil {
		return model.Session{}, fmt.Errorf("reading the session: %w", err)
	}

	return session, nil
}

// signIn signs in with the stored credentials, and replaces the stored session with the
// new one. Absent settings, a rejection, or a request for a new password remove the stored
// session, because no later call can renew it.
func (a *Account) signIn(ctx context.Context) (model.Session, error) {
	credentials, err := a.credentials(ctx)
	if errors.Is(err, errs.ErrCredentialsAbsent) {
		return model.Session{}, errors.Join(err, a.forgetSession(ctx))
	}

	if err != nil {
		return model.Session{}, err
	}

	session, err := a.client.SignIn(ctx, credentials.Email, credentials.Password)
	if IsRefusal(err) {
		return model.Session{}, errors.Join(err, a.forgetSession(ctx))
	}

	if err != nil {
		return model.Session{}, fmt.Errorf("signing in: %w", err)
	}

	stored, err := session.MarshalSecret()
	if err != nil {
		return model.Session{}, fmt.Errorf("storing the session: %w", err)
	}

	if err = a.secrets.Put(ctx, pluginapi.SecretScopeWorkspace, SecretSession, stored); err != nil {
		return model.Session{}, fmt.Errorf("storing the session: %w", err)
	}

	return session, nil
}

func (a *Account) forgetSession(ctx context.Context) error {
	if err := a.secrets.Delete(ctx, pluginapi.SecretScopeWorkspace, SecretSession); err != nil {
		return fmt.Errorf("removing the session: %w", err)
	}

	return nil
}
