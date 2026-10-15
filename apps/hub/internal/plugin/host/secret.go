package host

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// SecretStore reads and writes the plugin secrets of every plugin.
type SecretStore interface {
	ReadSecret(
		ctx context.Context,
		pluginID *pluginapi.PluginID,
		scope pluginapi.SecretScope,
		name string,
	) (string, error)
	// WriteSecret stores the value under the name. A zero expiresAt stores it with no expiry.
	WriteSecret(
		ctx context.Context,
		pluginID *pluginapi.PluginID,
		scope pluginapi.SecretScope,
		name string,
		value string,
		expiresAt time.Time,
	) error
	DeleteSecret(
		ctx context.Context,
		pluginID *pluginapi.PluginID,
		scope pluginapi.SecretScope,
		name string,
	) error
}

type boundSecrets struct {
	store    SecretStore
	pluginID *pluginapi.PluginID
}

func (s *boundSecrets) Get(
	ctx context.Context,
	scope pluginapi.SecretScope,
	name string,
) (string, error) {
	value, err := s.store.ReadSecret(ctx, s.pluginID, scope, name)
	if err != nil {
		return "", fmt.Errorf("reading the plugin secret %q: %w", name, err)
	}

	return value, nil
}

func (s *boundSecrets) Put(
	ctx context.Context,
	scope pluginapi.SecretScope,
	name string,
	value string,
) error {
	return s.write(ctx, scope, name, value, time.Time{})
}

func (s *boundSecrets) PutUntil(
	ctx context.Context,
	scope pluginapi.SecretScope,
	name string,
	value string,
	expiresAt time.Time,
) error {
	return s.write(ctx, scope, name, value, expiresAt)
}

func (s *boundSecrets) Delete(ctx context.Context, scope pluginapi.SecretScope, name string) error {
	if err := s.store.DeleteSecret(ctx, s.pluginID, scope, name); err != nil {
		return fmt.Errorf("deleting the plugin secret %q: %w", name, err)
	}

	return nil
}

func (s *boundSecrets) write(
	ctx context.Context,
	scope pluginapi.SecretScope,
	name string,
	value string,
	expiresAt time.Time,
) error {
	if err := s.store.WriteSecret(ctx, s.pluginID, scope, name, value, expiresAt); err != nil {
		return fmt.Errorf("writing the plugin secret %q: %w", name, err)
	}

	return nil
}
