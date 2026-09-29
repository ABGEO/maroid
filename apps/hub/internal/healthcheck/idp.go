package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hellofresh/health-go/v5"
)

// ErrUnexpectedStatus indicates that a dependency answered with a status other than 200.
var ErrUnexpectedStatus = errors.New("healthcheck: unexpected status")

// idpCheck checks that the identity provider serves its discovery document.
func idpCheck(client *http.Client, issuer string) health.Config {
	discoveryURL := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"

	return limited("idp", func(ctx context.Context) error {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
		if err != nil {
			return fmt.Errorf("creating the discovery request: %w", err)
		}

		response, err := client.Do(request)
		if err != nil {
			return fmt.Errorf("requesting the discovery document: %w", err)
		}
		defer func() { _ = response.Body.Close() }()

		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("%w: %d", ErrUnexpectedStatus, response.StatusCode)
		}

		return nil
	})
}
