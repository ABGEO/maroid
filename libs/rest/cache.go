package rest

import "net/http"

const (
	// NoStore is the period that every answer carries unless a handler says
	// otherwise.
	NoStore = "no-cache, no-store, must-revalidate, max-age=0"
	// Immutable is the period of an asset whose name carries a content hash. A
	// changed asset takes a new address, so no cache serves a stale one.
	Immutable = "public, max-age=31536000, immutable"

	cacheControlHeader = "Cache-Control"
)

// CachePeriod sets the default period on every answer. A handler that answers
// the same bytes to everyone sets the header itself, and this keeps what it set.
//
// The default runs first so that a route added later carries it without anyone
// deciding, which is the safe direction.
func CachePeriod(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(cacheControlHeader, NoStore)

		next.ServeHTTP(w, r)
	})
}
