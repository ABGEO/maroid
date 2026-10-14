package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/gwp/config"
	"github.com/abgeo/maroid/plugins/gwp/errs"
)

// SecretToken names the workspace plugin secret that holds the token of the water account.
const SecretToken = "token"

// account reads the credentials and the token of the acting workspace, and signs in.
// The connection service and the collector share it.
type account struct {
	client   APIClientService
	settings *pluginapi.PluginSettings
	secrets  *pluginapi.PluginSecrets
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

// token answers the stored token, or errs.ErrNotConnected when the workspace holds none.
func (a *account) token(ctx context.Context) (string, error) {
	token, err := a.secrets.Get(ctx, pluginapi.SecretScopeWorkspace, SecretToken)
	if errors.Is(err, pluginapi.ErrSecretAbsent) {
		return "", errs.ErrNotConnected
	}

	if err != nil {
		return "", fmt.Errorf("reading the token: %w", err)
	}

	return token, nil
}

// signIn signs in with the stored credentials, and replaces the stored token with
// the one that the water service gives. Credentials that the water service rejects
// remove the stored token, because no later call can renew it.
func (a *account) signIn(ctx context.Context) (string, error) {
	credentials, err := a.credentials(ctx)
	if err != nil {
		return "", err
	}

	token, err := a.client.Authenticate(ctx, credentials.Username, credentials.Password)
	if errors.Is(err, errs.ErrCredentialsRejected) {
		return "", errors.Join(err, a.forgetToken(ctx))
	}

	if err != nil {
		return "", fmt.Errorf("signing in: %w", err)
	}

	if err = a.secrets.Put(ctx, pluginapi.SecretScopeWorkspace, SecretToken, token); err != nil {
		return "", fmt.Errorf("storing the token: %w", err)
	}

	return token, nil
}

func (a *account) forgetToken(ctx context.Context) error {
	if err := a.secrets.Delete(ctx, pluginapi.SecretScopeWorkspace, SecretToken); err != nil {
		return fmt.Errorf("removing the token: %w", err)
	}

	return nil
}

// call runs the call with the token. A rejected token gives way to one new sign in,
// because the stored credentials need no person to send a code.
func (a *account) call(ctx context.Context, token string, run func(token string) error) error {
	err := run(token)
	if !errors.Is(err, errs.ErrTokenRejected) {
		return err
	}

	token, err = a.signIn(ctx)
	if err != nil {
		return fmt.Errorf("replacing a rejected token: %w", err)
	}

	err = run(token)
	if errors.Is(err, errs.ErrTokenRejected) {
		return fmt.Errorf("%w: the water service rejected a new token", errs.ErrRequestFailed)
	}

	return err
}
