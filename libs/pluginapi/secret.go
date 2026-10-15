package pluginapi

import (
	"context"
	"errors"
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

// PluginSecrets reads and writes the plugin secrets of one plugin, for the acting user
// and the acting workspace.
type PluginSecrets interface {
	// Get returns the plugin secret under the name. It returns ErrSecretAbsent when the
	// scope holds none.
	Get(ctx context.Context, scope SecretScope, name string) (string, error)
	// Put stores the plugin secret under the name, with no expiry. It replaces the
	// value and the expiry that the name holds.
	Put(ctx context.Context, scope SecretScope, name string, value string) error
	// PutUntil stores the plugin secret under the name until expiresAt. It replaces
	// the value and the expiry that the name holds.
	PutUntil(
		ctx context.Context,
		scope SecretScope,
		name string,
		value string,
		expiresAt time.Time,
	) error
	// Delete removes the plugin secret under the name. A name that the scope does not
	// hold is no failure.
	Delete(ctx context.Context, scope SecretScope, name string) error
}
