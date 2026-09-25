package rest

import (
	"context"
	"net/http"
	"strings"
)

const baseURLKey contextKey = 1

// BaseURL puts the external address of the deployment in the context of every
// request, so that a link comes from the stored value and never from the Host
// header that a caller sends.
func BaseURL(external string) func(http.Handler) http.Handler {
	trimmed := strings.TrimSuffix(external, "/")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(ContextWithBaseURL(r.Context(), trimmed)))
		})
	}
}

// ContextWithBaseURL returns a context that carries the external address.
func ContextWithBaseURL(ctx context.Context, external string) context.Context {
	return context.WithValue(ctx, baseURLKey, external)
}

// BaseURLFromContext returns the external address, or the empty string when the
// context carries none.
func BaseURLFromContext(ctx context.Context) string {
	external, _ := ctx.Value(baseURLKey).(string)

	return external
}
