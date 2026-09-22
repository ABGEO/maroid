package server

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/abgeo/maroid/libs/problem"
)

// routableMethods holds every method that a route of the hub can carry.
//
//nolint:gochecknoglobals // A fixed list that the Allow header of a 405 reads.
var routableMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}

// notFound answers a route that the router does not hold.
func notFound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		problem.Write(w, r, problem.NewNotFound())
	}
}

// methodNotAllowed answers a method that the route does not hold.
//
// It names the methods that the route does hold, because RFC 9110 asks a 405 for
// an Allow header. Chi builds that header in its own responder and drops it as
// soon as a router sets one of its own, so this asks the router for each method.
func methodNotAllowed(router *chi.Mux) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if allowed := allowedMethods(router, r.URL.Path); allowed != "" {
			w.Header().Set("Allow", allowed)
		}

		problem.Write(w, r, problem.NewMethodNotAllowed())
	}
}

// allowedMethods asks the router which methods the path holds.
func allowedMethods(router *chi.Mux, path string) string {
	methods := make([]string, 0, len(routableMethods))

	for _, method := range routableMethods {
		if router.Match(chi.NewRouteContext(), method, path) {
			methods = append(methods, method)
		}
	}

	return strings.Join(methods, ", ")
}

// recoverer answers a panic with a problem. The Recoverer of chi writes plain
// text, and every failure of the hub answers with a problem.
func recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer recoverPanic(logger, w, r)

			next.ServeHTTP(w, r)
		})
	}
}

// recoverPanic turns a panic into a problem. It passes http.ErrAbortHandler on,
// because the server reads that one to drop the connection without a log line.
func recoverPanic(logger *slog.Logger, w http.ResponseWriter, r *http.Request) {
	cause := recover()
	if cause == nil {
		return
	}

	if aborted, ok := cause.(error); ok && errors.Is(aborted, http.ErrAbortHandler) {
		panic(cause)
	}

	logger.ErrorContext(
		r.Context(),
		"the handler panicked",
		slog.Any("cause", cause),
		slog.String("stack", string(debug.Stack())),
	)

	problem.Write(w, r, problem.NewInternal())
}
