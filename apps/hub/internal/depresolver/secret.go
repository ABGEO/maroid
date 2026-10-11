package depresolver

import (
	"context"
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/openbao"
	"github.com/abgeo/maroid/apps/hub/internal/pluginsecret"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
)

// OpenBaoSession initializes and returns the session that every use of OpenBao shares.
func (c *Container) OpenBaoSession() (*openbao.Session, error) {
	c.openBaoSession.mu.Lock()
	defer c.openBaoSession.mu.Unlock()

	var err error

	c.openBaoSession.once.Do(func() {
		c.openBaoSession.instance, err = openbao.New(
			context.Background(),
			&c.Config().OpenBao,
			c.Logger(),
		)
	})

	if err != nil {
		c.openBaoSession.once = sync.Once{}

		return nil, fmt.Errorf("initializing the OpenBao session: %w", err)
	}

	return c.openBaoSession.instance, nil
}

// SecretCipher initializes and returns the cipher that protects a secret.
func (c *Container) SecretCipher() (secret.Cipher, error) {
	c.secretCipher.mu.Lock()
	defer c.secretCipher.mu.Unlock()

	var err error

	c.secretCipher.once.Do(func() {
		var session *openbao.Session

		session, err = c.OpenBaoSession()
		if err != nil {
			return
		}

		c.secretCipher.instance = secret.NewTransit(
			session.Client(),
			c.Config().OpenBao.TransitMount,
		)
	})

	if err != nil {
		c.secretCipher.once = sync.Once{}

		return nil, fmt.Errorf("initializing the secret cipher: %w", err)
	}

	return c.secretCipher.instance, nil
}

// PluginSecretStore initializes and returns the store of the plugin secrets.
func (c *Container) PluginSecretStore() (*pluginsecret.Store, error) {
	c.pluginSecretStore.mu.Lock()
	defer c.pluginSecretStore.mu.Unlock()

	var err error

	c.pluginSecretStore.once.Do(func() {
		var (
			dbInstance *sqlx.DB
			cipher     secret.Cipher
		)

		dbInstance, err = c.Database()
		if err != nil {
			return
		}

		cipher, err = c.SecretCipher()
		if err != nil {
			return
		}

		c.pluginSecretStore.instance = pluginsecret.NewStore(dbInstance, cipher)
	})

	if err != nil {
		c.pluginSecretStore.once = sync.Once{}

		return nil, fmt.Errorf("initializing the plugin secret store: %w", err)
	}

	return c.pluginSecretStore.instance, nil
}
