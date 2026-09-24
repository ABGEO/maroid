package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/libs/rest"
)

// AllowedNetworks returns a middleware that refuses a caller outside every
// given network.
func AllowedNetworks(
	logger *slog.Logger,
	networks []string,
) (func(http.Handler) http.Handler, error) {
	allowed, err := ParsePrefixes(networks)
	if err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := middleware.GetClientIPAddr(r.Context())
			if !ip.IsValid() {
				logger.WarnContext(r.Context(), "the request carries no client address")
				rest.Write(w, r, problems.NewNetworkNotAllowed())

				return
			}

			for _, network := range allowed {
				if network.Contains(ip) {
					next.ServeHTTP(w, r)

					return
				}
			}

			logger.WarnContext(
				r.Context(),
				"request from disallowed network",
				slog.String("client.ip", ip.String()),
			)
			rest.Write(w, r, problems.NewNetworkNotAllowed())
		})
	}, nil
}

// ParsePrefixes reads each CIDR of the configuration.
func ParsePrefixes(cidrs []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(cidrs))

	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			return nil, fmt.Errorf("parsing CIDR %q: %w", cidr, err)
		}

		prefixes = append(prefixes, prefix.Masked())
	}

	return prefixes, nil
}
