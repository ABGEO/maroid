package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/telmico/config"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

// SecretSession names the workspace plugin secret that holds the session of the portal.
const SecretSession = "session"

// account reads the credentials and the session of the acting workspace, and signs in.
// The connection service, the collector, the bills, and the documents share it.
type account struct {
	client   APIClientService
	settings pluginapi.PluginSettings
	secrets  pluginapi.PluginSecrets
}

func (a *account) credentials(ctx context.Context) (*config.WorkspaceSettings, error) {
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
func (a *account) session(ctx context.Context) (model.Session, error) {
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
// new one. Credentials that the portal rejects remove the stored session, because no
// later call can renew it.
func (a *account) signIn(ctx context.Context) (model.Session, error) {
	credentials, err := a.credentials(ctx)
	if err != nil {
		return model.Session{}, err
	}

	session, err := a.client.SignIn(ctx, credentials.Username, credentials.Password)
	if errors.Is(err, errs.ErrCredentialsRejected) {
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

func (a *account) forgetSession(ctx context.Context) error {
	if err := a.secrets.Delete(ctx, pluginapi.SecretScopeWorkspace, SecretSession); err != nil {
		return fmt.Errorf("removing the session: %w", err)
	}

	return nil
}

// callWith runs the call with the session. A rejected session gives way to one new sign
// in, because the stored credentials need no person to send a code. It answers the
// session that the next call of the same run takes.
func callWith[T any](
	ctx context.Context,
	acct *account,
	session model.Session,
	run func(model.Session) (T, error),
) (T, model.Session, error) {
	var none T

	answer, err := run(session)
	if !errors.Is(err, errs.ErrSessionRejected) {
		return answer, session, err
	}

	session, err = acct.signIn(ctx)
	if err != nil {
		return none, model.Session{}, fmt.Errorf("replacing a rejected session: %w", err)
	}

	answer, err = run(session)
	if errors.Is(err, errs.ErrSessionRejected) {
		return none, model.Session{}, fmt.Errorf(
			"%w: the portal rejected a new session",
			errs.ErrRequestFailed,
		)
	}

	return answer, session, err
}
