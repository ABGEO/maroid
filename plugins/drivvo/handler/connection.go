package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/drivvo/dto"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/permission"
	"github.com/abgeo/maroid/plugins/drivvo/service"
)

const pathConnection = "/connections/self"

// ConnectionService connects the car log account of the acting workspace.
type ConnectionService interface {
	Status(ctx context.Context) (*model.Connection, error)
	Connect(ctx context.Context) (*model.Connection, error)
	Disconnect(ctx context.Context) error
}

var _ ConnectionService = (*service.Connections)(nil)

// ConnectionHandler serves the connection page: it reads, starts, and ends the connection
// of the car log account of the acting workspace.
type ConnectionHandler struct {
	logger      *slog.Logger
	connections ConnectionService
}

// NewConnectionHandler creates a new ConnectionHandler.
func NewConnectionHandler(logger *slog.Logger, connections ConnectionService) *ConnectionHandler {
	return &ConnectionHandler{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "connection"),
		),
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
			Permission: permission.AccountRead,
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

// Connect handles PUT /connections/self. A repeat signs in again and leaves the workspace
// connected, so the method is PUT.
func (h *ConnectionHandler) Connect(w http.ResponseWriter, r *http.Request) {
	connection, err := h.connections.Connect(r.Context())
	if err != nil {
		writeFailure(w, r, h.logger, "connecting the car log account", err)

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
