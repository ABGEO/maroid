package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// PingHandler represents the Ping handler interface.
type PingHandler interface {
	Handler

	Ping(w http.ResponseWriter, r *http.Request) error
}

// Ping represents the Ping handler.
type Ping struct {
	logger *slog.Logger
}

var _ PingHandler = (*Ping)(nil)

// NewPing creates a new Ping handler.
func NewPing(logger *slog.Logger) *Ping {
	return &Ping{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "ping"),
		),
	}
}

// Register registers the Ping routes.
func (h *Ping) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	router.Get("/ping", Wrap(h.logger, h.Ping))
}

// Ping handles the /ping endpoint.
func (h *Ping) Ping(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]any{"message": "pong"})

	return nil
}
