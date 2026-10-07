package healthcheck

import (
	"context"
	"errors"
	"fmt"

	"github.com/hellofresh/health-go/v5"
	"github.com/openbao/openbao/api/v2"
)

var (
	// ErrOpenBaoSealed indicates that OpenBao is sealed and cannot serve a request.
	ErrOpenBaoSealed = errors.New("healthcheck: openbao is sealed")
	// ErrOpenBaoUninitialized indicates that OpenBao has not been initialized.
	ErrOpenBaoUninitialized = errors.New("healthcheck: openbao is not initialized")
)

// secretStoreCheck checks that the secret store is initialized and unsealed, and
// that it accepts the token of the hub.
func secretStoreCheck(client *api.Client) health.Config {
	return limited("secret-store", func(ctx context.Context) error {
		status, err := client.Sys().HealthWithContext(ctx)
		if err != nil {
			return fmt.Errorf("reading the openbao health: %w", err)
		}

		switch {
		case !status.Initialized:
			return ErrOpenBaoUninitialized
		case status.Sealed:
			return ErrOpenBaoSealed
		}

		// The health of the store needs no token, so it stays up after the token of
		// the hub ends.
		if _, err = client.Auth().Token().LookupSelfWithContext(ctx); err != nil {
			return fmt.Errorf("reading the openbao token: %w", err)
		}

		return nil
	})
}
