package service

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/magti/config"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
)

// SecretSession names the workspace plugin secret that holds the session of the portal.
const SecretSession = "session"

// Account reads the credentials and the session of the acting workspace, signs in, and
// orders the calls of each workspace. The connections, the states, and the collector
// share one Account, because the portal keeps the selected subscriber in the session.
type Account struct {
	client   APIClientService
	settings pluginapi.PluginSettings
	secrets  pluginapi.PluginSecrets

	mu    sync.Mutex
	locks map[string]chan struct{}
}

// NewAccount creates a new Account.
func NewAccount(
	client APIClientService,
	settings pluginapi.PluginSettings,
	secrets pluginapi.PluginSecrets,
) *Account {
	return &Account{
		client:   client,
		settings: settings,
		secrets:  secrets,
		locks:    map[string]chan struct{}{},
	}
}

// lock waits until no other call of the acting workspace holds the session, or until
// the context ends. The answered function releases it.
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
		return nil, fmt.Errorf("%w: waiting for the session: %w", errs.ErrRequestFailed, ctx.Err())
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

	if credentials.Username == "" || credentials.Password == "" {
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
// new one. A refusal of the sign in removes the stored session, because no later call
// can renew it.
func (a *Account) signIn(ctx context.Context) (model.Session, error) {
	credentials, err := a.credentials(ctx)
	if err != nil {
		return model.Session{}, err
	}

	session, err := a.client.SignIn(ctx, credentials.Username, credentials.Password)
	if isRefusal(err) {
		return model.Session{}, errors.Join(err, a.forgetSession(ctx))
	}

	if err != nil {
		return model.Session{}, fmt.Errorf("signing in: %w", err)
	}

	if err = a.storeSession(ctx, session); err != nil {
		return model.Session{}, err
	}

	return session, nil
}

func (a *Account) storeSession(ctx context.Context, session model.Session) error {
	stored, err := session.MarshalSecret()
	if err != nil {
		return fmt.Errorf("storing the session: %w", err)
	}

	if err = a.secrets.Put(ctx, pluginapi.SecretScopeWorkspace, SecretSession, stored); err != nil {
		return fmt.Errorf("storing the session: %w", err)
	}

	return nil
}

func (a *Account) forgetSession(ctx context.Context) error {
	if err := a.secrets.Delete(ctx, pluginapi.SecretScopeWorkspace, SecretSession); err != nil {
		return fmt.Errorf("removing the session: %w", err)
	}

	return nil
}

// isRefusal reports a sign in that the portal refused for the account, as against one
// that failed on the way.
func isRefusal(err error) bool {
	return errors.Is(err, errs.ErrCredentialsRejected) ||
		errors.Is(err, errs.ErrCodeRequired) ||
		errors.Is(err, errs.ErrAccountLocked)
}

// callLocked holds the session of the acting workspace, and runs the call with the
// stored session. A rejected session gives way to one new sign in, and the call runs
// again from its start, so a call that selects a subscriber selects it again. A rejected
// session that no sign in can renew, for want of the settings, goes too.
func callLocked[T any](
	ctx context.Context,
	acct *Account,
	run func(model.Session) (T, error),
) (T, error) {
	var none T

	unlock, err := acct.lock(ctx)
	if err != nil {
		return none, err
	}
	defer unlock()

	session, err := acct.session(ctx)
	if err != nil {
		return none, err
	}

	answer, err := runKeeping(ctx, acct, session, run)
	if !errors.Is(err, errs.ErrSessionRejected) {
		return answer, err
	}

	session, err = acct.signIn(ctx)
	if errors.Is(err, errs.ErrCredentialsAbsent) {
		err = errors.Join(err, acct.forgetSession(ctx))
	}

	if err != nil {
		return none, fmt.Errorf("replacing a rejected session: %w", err)
	}

	answer, err = runKeeping(ctx, acct, session, run)
	if errors.Is(err, errs.ErrSessionRejected) {
		return none, fmt.Errorf("%w: the portal rejected a new session", errs.ErrRequestFailed)
	}

	return answer, err
}

// runKeeping runs the call, and stores the session again when a page of the call set a
// new cookie. The client adds such a cookie to the map of the session it was given.
func runKeeping[T any](
	ctx context.Context,
	acct *Account,
	session model.Session,
	run func(model.Session) (T, error),
) (T, error) {
	before := maps.Clone(session.Cookies)

	answer, err := run(session)
	if err != nil || maps.Equal(before, session.Cookies) {
		return answer, err
	}

	if err = acct.storeSession(ctx, session); err != nil {
		var none T

		return none, err
	}

	return answer, nil
}
