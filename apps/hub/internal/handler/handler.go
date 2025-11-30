// Package handler provides HTTP handlers and utilities.
package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler represents an HTTP handler.
type Handler interface {
	Register(router chi.Router)
}

// Fn represents a handler function that returns an error.
type Fn func(http.ResponseWriter, *http.Request) error

// RegisterHandlers registers multiple handlers to the given router.
func RegisterHandlers(router chi.Router, handlers ...Handler) {
	for _, h := range handlers {
		h.Register(router)
	}
}

// Wrap wraps a handler function with logging for errors.
func Wrap(logger *slog.Logger, fn Fn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := fn(w, r.WithContext(ctx))
		if err != nil {
			logger.ErrorContext(ctx, "handler errored", slog.Any("error", err))

			return
		}
	}
}
