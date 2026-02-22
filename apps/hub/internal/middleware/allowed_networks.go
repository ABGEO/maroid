package middleware

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/go-chi/render"
)

// AllowedNetworks returns a Chi HTTP middleware that rejects requests
// from IPs not contained in any of the given CIDR networks.
func AllowedNetworks(
	logger *slog.Logger,
	networks []string,
) (func(http.Handler) http.Handler, error) {
	parsed := make([]*net.IPNet, 0, len(networks))

	for _, cidr := range networks {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("parsing CIDR %q: %w", cidr, err)
		}

		parsed = append(parsed, ipNet)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}

			ip := net.ParseIP(host)
			if ip == nil {
				logger.Warn("could not parse remote IP", slog.String("remote_addr", r.RemoteAddr))
				renderError(w, r)

				return
			}

			for _, ipNet := range parsed {
				if ipNet.Contains(ip) {
					next.ServeHTTP(w, r)

					return
				}
			}

			logger.Warn("request from disallowed network", slog.String("remote_ip", ip.String()))
			renderError(w, r)
		})
	}, nil
}

func renderError(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusForbidden)
	render.JSON(w, r, map[string]any{
		"message": "Forbidden",
	})
}
