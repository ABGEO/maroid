package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Discoverer checks that an issuer publishes an OIDC discovery document.
type Discoverer interface {
	Discover(ctx context.Context, issuer string) error
}

// OIDCDiscovery reads the discovery document with go-oidc, which also checks that the
// issuer of the document matches.
type OIDCDiscovery struct {
	Timeout time.Duration
}

var _ Discoverer = OIDCDiscovery{}

// Discover reads the discovery document of the issuer.
func (d OIDCDiscovery) Discover(ctx context.Context, issuer string) error {
	ctx, cancel := context.WithTimeout(ctx, d.Timeout)
	defer cancel()

	if _, err := oidc.NewProvider(ctx, issuer); err != nil {
		return fmt.Errorf("reading the discovery document of %s: %w", issuer, err)
	}

	return nil
}
