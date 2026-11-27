package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/dto"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/permission"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

const pathConnection = "/connections/self"

// ConnectionService connects the pension account of the acting user.
type ConnectionService interface {
	Status(ctx context.Context) (*model.Connection, error)
	Connect(ctx context.Context) (*model.Connection, error)
	Disconnect(ctx context.Context) error
}

var _ ConnectionService = (*service.Connections)(nil)

// ConnectionHandler serves the connection page: it reads, starts, and ends the
// connection of the pension account of the acting user.
type ConnectionHandler struct {
	logger      *slog.Logger
	connections ConnectionService
}

// NewConnectionHandler creates a new ConnectionHandler.
func NewConnectionHandler(
	logger *slog.Logger,
	connections ConnectionService,
) *ConnectionHandler {
	return &ConnectionHandler{
		logger:      loggerOf(logger, "connection"),
		connections: connections,
	}
}

// Routes returns the HTTP routes of the connection page.
func (h *ConnectionHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathConnection,
			Handler:    h.Read,
			Permission: permission.AccountConnect,
		},
		{
			Method:     http.MethodPut,
			Pattern:    pathConnection,
			Handler:    h.Connect,
			Permission: permission.AccountConnect,
		},
		{
			Method:     http.MethodDelete,
			Pattern:    pathConnection,
			Handler:    h.Disconnect,
			Permission: permission.AccountConnect,
		},
	}
}

// Read handles GET /connections/self.
func (h *ConnectionHandler) Read(w http.ResponseWriter, r *http.Request) {
	connection, err := h.connections.Status(r.Context())
	if err != nil {
		writeFailure(w, r, h.logger, "reading the connection", err)

		return
	}

	writeOK(w, r, h.logger, dto.NewConnectionResponse(connection))
}

// Connect handles PUT /connections/self. A repeat signs in again and leaves the user
// connected, so the method is PUT.
func (h *ConnectionHandler) Connect(w http.ResponseWriter, r *http.Request) {
	connection, err := h.connections.Connect(r.Context())
	if err != nil {
		writeFailure(w, r, h.logger, "connecting the pension account", err)

		return
	}

	writeOK(w, r, h.logger, dto.NewConnectionResponse(connection))
}

// Disconnect handles DELETE /connections/self.
func (h *ConnectionHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	if err := h.connections.Disconnect(r.Context()); err != nil {
		writeFailure(w, r, h.logger, "ending the connection", err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeOK answers 200 with the body. The status line is gone when the encode fails,
// so the failure goes to the log alone.
func writeOK(w http.ResponseWriter, r *http.Request, logger *slog.Logger, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		logger.ErrorContext(r.Context(), "failed to encode the answer", slog.Any("error", err))
	}
}
