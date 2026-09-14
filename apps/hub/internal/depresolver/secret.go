package depresolver

import (
	"context"
	"fmt"
	"sync"

	"github.com/openbao/openbao/api/v2"

	"github.com/abgeo/maroid/apps/hub/internal/openbao"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
)

// OpenBaoClient initializes and returns the client that every use of OpenBao shares.
func (c *Container) OpenBaoClient() (*api.Client, error) {
	c.openBaoClient.mu.Lock()
	defer c.openBaoClient.mu.Unlock()

	var err error

	c.openBaoClient.once.Do(func() {
		c.openBaoClient.instance, err = openbao.New(context.Background(), &c.Config().OpenBao)
	})

	if err != nil {
		c.openBaoClient.once = sync.Once{}

		return nil, fmt.Errorf("initializing the OpenBao client: %w", err)
	}

	return c.openBaoClient.instance, nil
}

// SecretCipher initializes and returns the cipher that protects a secret.
func (c *Container) SecretCipher() (secret.Cipher, error) {
	c.secretCipher.mu.Lock()
	defer c.secretCipher.mu.Unlock()

	var err error

	c.secretCipher.once.Do(func() {
		var client *api.Client

		client, err = c.OpenBaoClient()
		if err != nil {
			return
		}

		c.secretCipher.instance = secret.NewTransit(client, c.Config().OpenBao.TransitMount)
	})

	if err != nil {
		c.secretCipher.once = sync.Once{}

		return nil, fmt.Errorf("initializing the secret cipher: %w", err)
	}

	return c.secretCipher.instance, nil
}
