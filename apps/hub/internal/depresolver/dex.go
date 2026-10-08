package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/dex"
	"github.com/abgeo/maroid/apps/hub/internal/provider"
)

// DexClient initializes and returns the client of the gRPC API of Dex.
func (c *Container) DexClient() (*dex.GRPC, error) {
	c.dexClient.mu.Lock()
	defer c.dexClient.mu.Unlock()

	var err error

	c.dexClient.once.Do(func() {
		c.dexClient.instance, err = dex.New(&c.Config().Dex)
	})

	if err != nil {
		c.dexClient.once = sync.Once{}

		return nil, fmt.Errorf("initializing the Dex client: %w", err)
	}

	return c.dexClient.instance, nil
}

// CloseDexClient closes the connection to Dex.
func (c *Container) CloseDexClient() error {
	if c.dexClient.instance == nil {
		return nil
	}

	if err := c.dexClient.instance.Close(); err != nil {
		return fmt.Errorf("closing the Dex client: %w", err)
	}

	return nil
}

// ProviderService initializes and returns the service of the providers.
func (c *Container) ProviderService() (*provider.Manager, error) {
	c.providerService.mu.Lock()
	defer c.providerService.mu.Unlock()

	var err error

	c.providerService.once.Do(func() {
		var client *dex.GRPC

		client, err = c.DexClient()
		if err != nil {
			return
		}

		c.providerService.instance = provider.NewManager(client, c.Config().OIDC.Issuer)
	})

	if err != nil {
		c.providerService.once = sync.Once{}

		return nil, fmt.Errorf("initializing the provider service: %w", err)
	}

	return c.providerService.instance, nil
}
