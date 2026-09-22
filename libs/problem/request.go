package problem

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// RequestIDHeader carries the identifier of the request on every response.
const RequestIDHeader = "X-Request-Id"

// instancePrefix turns the identifier of a request into the URI that RFC 9457
// asks for in the instance member.
const instancePrefix = "urn:maroid:request:"

type contextKey int

const requestIDKey contextKey = 0

// RequestID gives each request an identifier, puts it in the context, and sets it
// on the response.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identifier, err := uuid.NewV7()
		if err != nil {
			next.ServeHTTP(w, r)

			return
		}

		value := identifier.String()
		w.Header().Set(RequestIDHeader, value)

		next.ServeHTTP(w, r.WithContext(ContextWithRequestID(r.Context(), value)))
	})
}

// ContextWithRequestID returns a context that carries the identifier of the request.
func ContextWithRequestID(ctx context.Context, identifier string) context.Context {
	return context.WithValue(ctx, requestIDKey, identifier)
}

// RequestIDFromContext returns the identifier of the request, or the empty string
// when the context carries none. The value is the bare UUID that the log holds.
func RequestIDFromContext(ctx context.Context) string {
	identifier, _ := ctx.Value(requestIDKey).(string)

	return identifier
}

// Instance returns the value of the instance member for one identifier.
func Instance(identifier string) string {
	if identifier == "" {
		return ""
	}

	return instancePrefix + identifier
}
