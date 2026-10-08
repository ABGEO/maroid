package depresolver

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"

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

		var database *sqlx.DB

		database, err = c.Database()
		if err != nil {
			return
		}

		cfg := c.Config()
		c.providerService.instance = provider.NewManager(
			client,
			database,
			provider.Settings{
				Issuer:        cfg.OIDC.Issuer,
				TelegramBotID: telegramBotID(cfg.Telegram.Token),
				Discoverer:    provider.OIDCDiscovery{Timeout: cfg.Dex.Timeout},
			},
		)
	})

	if err != nil {
		c.providerService.once = sync.Once{}

		return nil, fmt.Errorf("initializing the provider service: %w", err)
	}

	return c.providerService.instance, nil
}

// LocalAccounts initializes and returns the service of the local accounts.
func (c *Container) LocalAccounts() (*provider.Accounts, error) {
	c.localAccounts.mu.Lock()
	defer c.localAccounts.mu.Unlock()

	var err error

	c.localAccounts.once.Do(func() {
		c.localAccounts.instance, err = c.buildLocalAccounts()
	})

	if err != nil {
		c.localAccounts.once = sync.Once{}

		return nil, fmt.Errorf("initializing the local accounts: %w", err)
	}

	return c.localAccounts.instance, nil
}

func (c *Container) buildLocalAccounts() (*provider.Accounts, error) {
	client, err := c.DexClient()
	if err != nil {
		return nil, err
	}

	database, err := c.Database()
	if err != nil {
		return nil, err
	}

	providers, err := c.ProviderService()
	if err != nil {
		return nil, err
	}

	return provider.NewAccounts(client, database, providers), nil
}

// telegramBotID answers the numeric prefix of a bot token, which is the identifier of
// the bot.
func telegramBotID(token string) string {
	id, _, _ := strings.Cut(token, ":")

	return id
}
