package pluginapi

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// SecretScope names the owner of a plugin secret.
type SecretScope string

const (
	// SecretScopeUser gives the plugin secret to the acting user, in every workspace.
	SecretScopeUser SecretScope = "user"
	// SecretScopeWorkspace gives the plugin secret to the acting workspace, for every member.
	SecretScopeWorkspace SecretScope = "workspace"
)

var (
	// ErrSecretAbsent reports that the scope holds no plugin secret under the name,
	// or that its expiry passed.
	ErrSecretAbsent = errors.New("plugin secret: absent")
	// ErrSecretInvalid reports a name, a value, an expiry, or a scope that the store refuses.
	ErrSecretInvalid = errors.New("plugin secret: invalid")
	// ErrSecretScopeAbsent reports that the unit of work carries no owner of the scope:
	// no acting user for the user scope, or no acting workspace for the workspace scope.
	ErrSecretScopeAbsent = errors.New("plugin secret: no owner of the scope")
)

// SecretProvider reads and writes the plugin secrets of the acting user and of the
// acting workspace. The hub implements it, and Host hands it to a plugin.
type SecretProvider interface {
	ReadSecret(
		ctx context.Context,
		pluginID *PluginID,
		scope SecretScope,
		name string,
	) (string, error)
	// WriteSecret stores the value under the name. A zero expiresAt stores it with no expiry.
	WriteSecret(
		ctx context.Context,
		pluginID *PluginID,
		scope SecretScope,
		name string,
		value string,
		expiresAt time.Time,
	) error
	DeleteSecret(ctx context.Context, pluginID *PluginID, scope SecretScope, name string) error
}

// PluginSecrets binds a SecretProvider to one plugin.
type PluginSecrets struct {
	provider SecretProvider
	pluginID *PluginID
}

// NewPluginSecrets creates a new PluginSecrets instance.
func NewPluginSecrets(provider SecretProvider, pluginID *PluginID) *PluginSecrets {
	return &PluginSecrets{
		provider: provider,
		pluginID: pluginID,
	}
}

// Get returns the plugin secret under the name. It returns ErrSecretAbsent when the
// scope holds none.
func (s *PluginSecrets) Get(ctx context.Context, scope SecretScope, name string) (string, error) {
	value, err := s.provider.ReadSecret(ctx, s.pluginID, scope, name)
	if err != nil {
		return "", fmt.Errorf("reading the plugin secret %q: %w", name, err)
	}

	return value, nil
}

// Put stores the plugin secret under the name, with no expiry. It replaces the
// value and the expiry that the name holds.
func (s *PluginSecrets) Put(
	ctx context.Context,
	scope SecretScope,
	name string,
	value string,
) error {
	return s.write(ctx, scope, name, value, time.Time{})
}

// PutUntil stores the plugin secret under the name until expiresAt. It replaces the
// value and the expiry that the name holds.
func (s *PluginSecrets) PutUntil(
	ctx context.Context,
	scope SecretScope,
	name string,
	value string,
	expiresAt time.Time,
) error {
	return s.write(ctx, scope, name, value, expiresAt)
}

// Delete removes the plugin secret under the name. A name that the scope does not
// hold is no failure.
func (s *PluginSecrets) Delete(ctx context.Context, scope SecretScope, name string) error {
	if err := s.provider.DeleteSecret(ctx, s.pluginID, scope, name); err != nil {
		return fmt.Errorf("deleting the plugin secret %q: %w", name, err)
	}

	return nil
}

func (s *PluginSecrets) write(
	ctx context.Context,
	scope SecretScope,
	name string,
	value string,
	expiresAt time.Time,
) error {
	if err := s.provider.WriteSecret(ctx, s.pluginID, scope, name, value, expiresAt); err != nil {
		return fmt.Errorf("writing the plugin secret %q: %w", name, err)
	}

	return nil
}
