// Package address carries the external address of the deployment, from which
// every link is built.
package address

import (
	"context"
	"net/http"
	"strings"
)

type contextKey int

const baseURLKey contextKey = 0

// Middleware puts the external address of the deployment in the context of every
// request, so that a link comes from the stored value and never from the Host
// header that a caller sends.
func Middleware(external string) func(http.Handler) http.Handler {
	trimmed := strings.TrimSuffix(external, "/")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(withBase(r.Context(), trimmed)))
		})
	}
}

// withBase returns a context that carries the external address.
func withBase(ctx context.Context, external string) context.Context {
	return context.WithValue(ctx, baseURLKey, external)
}

// BaseFromContext returns the external address, or the empty string when the
// context carries none.
func BaseFromContext(ctx context.Context) string {
	external, _ := ctx.Value(baseURLKey).(string)

	return external
}
