package rest

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// FlowIDHeader carries the flow identifier, on the request that a caller sends
// and on every answer.
const FlowIDHeader = "X-Flow-ID"

const (
	instancePrefix  = "/flows/"
	flowIDMaxLength = 128
	flowIDExtra     = "/+_=-"
)

type contextKey int

const flowIDKey contextKey = 0

// FlowID gives each request a flow identifier, puts it in the context, and sets
// it on the response. It reads the one that the request carries, so a caller
// picks the value that every record of its request holds, and it makes a UUID
// version 7 when the request carries none.
func FlowID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//nolint:canonicalheader
		value := boundFlowID(r.Header.Get(FlowIDHeader))
		if value == "" {
			identifier, err := uuid.NewV7()
			if err != nil {
				next.ServeHTTP(w, r)

				return
			}

			value = identifier.String()
		}

		//nolint:canonicalheader
		w.Header().Set(FlowIDHeader, value)

		next.ServeHTTP(w, r.WithContext(ContextWithFlowID(r.Context(), value)))
	})
}

// boundFlowID drops every character that a flow identifier may not hold and cuts
// the result to the length that it may not pass. It answers the empty string
// when nothing remains, and the caller then makes its own value.
func boundFlowID(value string) string {
	bounded := strings.Map(func(character rune) rune {
		if flowIDRune(character) {
			return character
		}

		return -1
	}, value)

	if len(bounded) > flowIDMaxLength {
		return bounded[:flowIDMaxLength]
	}

	return bounded
}

func flowIDRune(character rune) bool {
	switch {
	case character >= 'a' && character <= 'z',
		character >= 'A' && character <= 'Z',
		character >= '0' && character <= '9':
		return true
	default:
		return strings.ContainsRune(flowIDExtra, character)
	}
}

// ContextWithFlowID returns a context that carries the flow identifier.
func ContextWithFlowID(ctx context.Context, identifier string) context.Context {
	return context.WithValue(ctx, flowIDKey, identifier)
}

// FlowIDFromContext returns the flow identifier, or the empty string when the
// context carries none. The value is bare, without the prefix of an instance.
func FlowIDFromContext(ctx context.Context) string {
	identifier, _ := ctx.Value(flowIDKey).(string)

	return identifier
}

// Instance returns the value of the instance member for one flow identifier.
func Instance(identifier string) string {
	if identifier == "" {
		return ""
	}

	return instancePrefix + identifier
}
