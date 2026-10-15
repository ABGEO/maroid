package pluginsecret

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// Store reads and writes the plugin secrets of the acting user and of the acting
// workspace. It calls the cipher outside the transaction, so no connection waits on it.
type Store struct {
	db     *sqlx.DB
	cipher secret.Cipher
}

// NewStore creates a new Store instance.
func NewStore(db *sqlx.DB, cipher secret.Cipher) *Store {
	return &Store{
		db:     db,
		cipher: cipher,
	}
}

// ReadSecret returns the plugin secret under the name. It returns
// pluginapi.ErrSecretAbsent when the scope holds none, or when its expiry passed.
func (s *Store) ReadSecret(
	ctx context.Context,
	pluginID *pluginapi.PluginID,
	scope pluginapi.SecretScope,
	name string,
) (string, error) {
	key, err := ownerKey(ctx, scope)
	if err != nil {
		return "", err
	}

	if err = validateName(name); err != nil {
		return "", err
	}

	entity, err := database.FetchScopeTx(ctx, s.db, func(tx *sqlx.Tx) (*model.PluginSecret, error) {
		return repository.NewPluginSecret(tx).Get(ctx, scope, pluginID.String(), name)
	})
	if err != nil {
		return "", fmt.Errorf("reading the plugin secret %q of the %s: %w", name, scope, err)
	}

	if entity == nil {
		return "", pluginapi.ErrSecretAbsent
	}

	value, err := s.cipher.Decrypt(ctx, key, entity.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("revealing the plugin secret %q of the %s: %w", name, scope, err)
	}

	return value, nil
}

// WriteSecret stores the value under the name, and replaces the value and the expiry
// that the name holds. A zero expiresAt stores the value with no expiry.
func (s *Store) WriteSecret(
	ctx context.Context,
	pluginID *pluginapi.PluginID,
	scope pluginapi.SecretScope,
	name string,
	value string,
	expiresAt time.Time,
) error {
	key, err := ownerKey(ctx, scope)
	if err != nil {
		return err
	}

	if err = validateWrite(name, value, expiresAt); err != nil {
		return err
	}

	ciphertext, err := s.cipher.Encrypt(ctx, key, value)
	if err != nil {
		return fmt.Errorf("protecting the plugin secret %q of the %s: %w", name, scope, err)
	}

	err = database.WithScopeTx(ctx, s.db, func(tx *sqlx.Tx) error {
		return repository.NewPluginSecret(tx).
			Upsert(ctx, scope, pluginID.String(), name, ciphertext, expiryOf(expiresAt))
	})
	if err != nil {
		return fmt.Errorf("writing the plugin secret %q of the %s: %w", name, scope, err)
	}

	return nil
}

// DeleteSecret removes the plugin secret under the name. A name that the scope does
// not hold is no failure.
func (s *Store) DeleteSecret(
	ctx context.Context,
	pluginID *pluginapi.PluginID,
	scope pluginapi.SecretScope,
	name string,
) error {
	if _, err := ownerKey(ctx, scope); err != nil {
		return err
	}

	if err := validateName(name); err != nil {
		return err
	}

	err := database.WithScopeTx(ctx, s.db, func(tx *sqlx.Tx) error {
		return repository.NewPluginSecret(tx).Delete(ctx, scope, pluginID.String(), name)
	})
	if err != nil {
		return fmt.Errorf("deleting the plugin secret %q of the %s: %w", name, scope, err)
	}

	return nil
}

// ownerKey returns the key that protects the owner of the scope. The policy of the
// table would hide every row from a unit of work with no owner, and a read would then
// report a value as absent, so the store refuses the call before it opens a transaction.
func ownerKey(ctx context.Context, scope pluginapi.SecretScope) (secret.Key, error) {
	switch scope {
	case pluginapi.SecretScopeUser:
		if userID := pluginapi.ActingUserFromContext(ctx); userID != "" {
			return secret.UserKey(userID), nil
		}
	case pluginapi.SecretScopeWorkspace:
		if workspaceID := pluginapi.ActingWorkspaceFromContext(ctx); workspaceID != "" {
			return secret.WorkspaceKey(workspaceID), nil
		}
	default:
		return "", fmt.Errorf("%w: the scope %q", pluginapi.ErrSecretInvalid, scope)
	}

	return "", fmt.Errorf("%w: the %s", pluginapi.ErrSecretScopeAbsent, scope)
}

func validateWrite(name string, value string, expiresAt time.Time) error {
	if err := validateName(name); err != nil {
		return err
	}

	if err := validateValue(value); err != nil {
		return err
	}

	return validateExpiry(expiresAt, time.Now())
}

func expiryOf(expiresAt time.Time) *time.Time {
	if expiresAt.IsZero() {
		return nil
	}

	return &expiresAt
}
