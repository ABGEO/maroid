package openbao

import (
	"context"
	"fmt"

	"github.com/openbao/openbao/api/auth/approle/v2"
	"github.com/openbao/openbao/api/v2"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

// New creates an OpenBao client and logs in with AppRole.
func New(ctx context.Context, cfg *config.OpenBao) (*api.Client, error) {
	clientConfig := api.DefaultConfig()
	clientConfig.Address = cfg.Address

	client, err := api.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("building the OpenBao client: %w", err)
	}

	login, err := approle.NewAppRoleAuth(
		cfg.RoleID,
		&approle.SecretID{FromString: cfg.SecretID},
	)
	if err != nil {
		return nil, fmt.Errorf("building the AppRole login: %w", err)
	}

	if _, err = client.Auth().Login(ctx, login); err != nil {
		return nil, fmt.Errorf("logging in to OpenBao: %w", err)
	}

	return client, nil
}
