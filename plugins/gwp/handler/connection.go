package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/gwp/dto"
	"github.com/abgeo/maroid/plugins/gwp/permission"
	"github.com/abgeo/maroid/plugins/gwp/service"
)

const (
	pathConnection = "/connections/self"
	pathAssignment = "/customer-assignments/self"

	maxAssignmentBody    = 4 << 10
	maxCustomerNumberLen = 64
)

// ConnectionHandler serves the connection page: it reads, starts, and ends the
// connection of the water account of the acting workspace, and assigns a customer.
type ConnectionHandler struct {
	logger      *slog.Logger
	connections service.ConnectionService
}

// NewConnectionHandler creates a new ConnectionHandler.
func NewConnectionHandler(
	logger *slog.Logger,
	connections service.ConnectionService,
) *ConnectionHandler {
	return &ConnectionHandler{
		logger:      logger,
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
		{
			Method:     http.MethodPut,
			Pattern:    pathAssignment,
			Handler:    h.Assign,
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

// Connect handles PUT /connections/self. A repeat signs in again and leaves the
// workspace connected, so the method is PUT.
func (h *ConnectionHandler) Connect(w http.ResponseWriter, r *http.Request) {
	connection, err := h.connections.Connect(r.Context())
	if err != nil {
		writeFailure(w, r, h.logger, "connecting the water account", err)

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

// Assign handles PUT /customer-assignments/self.
func (h *ConnectionHandler) Assign(w http.ResponseWriter, r *http.Request) {
	var request dto.AssignmentRequest
	if !readAssignment(w, r, &request) {
		return
	}

	connection, err := h.connections.Assign(r.Context(), request.CustomerNumber)
	if err != nil {
		writeFailure(w, r, h.logger, "assigning the customer", err)

		return
	}

	writeOK(w, r, h.logger, dto.NewConnectionResponse(connection))
}

// readAssignment decodes the body and checks the customer number. It writes the
// problem and answers false when the body is refused.
func readAssignment(w http.ResponseWriter, r *http.Request, target *dto.AssignmentRequest) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxAssignmentBody))
	decoder.DisallowUnknownFields()

	err := decoder.Decode(target)

	var tooLarge *http.MaxBytesError

	switch {
	case err == nil:
	case errors.As(err, &tooLarge):
		problem.Write(
			w,
			r,
			problem.NewContentTooLarge().WithDetail("The body holds at most 4 KiB."),
		)

		return false
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		problem.Write(w, r, problem.NewMemberUnknown().WithDetail(
			"The body carries "+strings.TrimPrefix(err.Error(), "json: unknown field ")+".",
		))

		return false
	default:
		problem.Write(w, r, problem.NewBodyInvalid())

		return false
	}

	length := utf8.RuneCountInString(target.CustomerNumber)
	if strings.TrimSpace(target.CustomerNumber) == "" || length > maxCustomerNumberLen {
		problem.Write(w, r, problem.NewValidationFailed(problem.FieldFailure{
			Detail:  "must hold 1 to 64 characters",
			Pointer: "#/customer_number",
		}))

		return false
	}

	return true
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
