package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/parking/dto"
	"github.com/abgeo/maroid/plugins/parking/permission"
	"github.com/abgeo/maroid/plugins/parking/service"
)

const (
	pathConnection   = "/connections/self"
	pathPending      = "/pending-connections"
	selfSegment      = "/self"
	pathCodeRequests = pathPending + selfSegment + "/code-requests"
	pathVerification = pathPending + selfSegment + "/verifications"

	minCodeLength = 4
	maxCodeLength = 8
)

// ConnectionHandler serves the connection page: it reads, starts, verifies, and ends
// the connection of the parking account of the acting user.
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

// Routes returns the HTTP routes of the connection.
func (h *ConnectionHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathConnection,
			Handler:    h.Read,
			Permission: permission.AccountConnect,
		},
		{
			Method:     http.MethodDelete,
			Pattern:    pathConnection,
			Handler:    h.Disconnect,
			Permission: permission.AccountConnect,
		},
		{
			Method:     http.MethodPost,
			Pattern:    pathPending,
			Handler:    h.Start,
			Permission: permission.AccountConnect,
		},
		{
			Method:     http.MethodPost,
			Pattern:    pathCodeRequests,
			Handler:    h.ResendCode,
			Permission: permission.AccountConnect,
		},
		{
			Method:     http.MethodPost,
			Pattern:    pathVerification,
			Handler:    h.Verify,
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

	h.writeJSON(w, r, http.StatusOK, dto.NewConnectionResponse(connection))
}

// Disconnect handles DELETE /connections/self.
func (h *ConnectionHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	if err := h.connections.Disconnect(r.Context()); err != nil {
		writeFailure(w, r, h.logger, "ending the connection", err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Start handles POST /pending-connections. Each call sends a new code, so the method
// is POST, and Location names the pending connection.
func (h *ConnectionHandler) Start(w http.ResponseWriter, r *http.Request) {
	pending, err := h.connections.Start(r.Context())
	if err != nil {
		writeFailure(w, r, h.logger, "starting a connection", err)

		return
	}

	w.Header().Set("Location", r.URL.Path+selfSegment)
	h.writeJSON(w, r, http.StatusCreated, dto.NewPendingConnectionResponse(pending))
}

// ResendCode handles POST /pending-connections/self/code-requests.
func (h *ConnectionHandler) ResendCode(w http.ResponseWriter, r *http.Request) {
	pending, err := h.connections.ResendCode(r.Context())
	if err != nil {
		writeFailure(w, r, h.logger, "requesting a new code", err)

		return
	}

	h.writeJSON(w, r, http.StatusOK, dto.NewPendingConnectionResponse(pending))
}

// Verify handles POST /pending-connections/self/verifications.
func (h *ConnectionHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var request dto.VerificationRequest
	if !readVerification(w, r, &request) {
		return
	}

	connection, err := h.connections.Verify(r.Context(), request.Code)
	if err != nil {
		writeFailure(w, r, h.logger, "verifying the code", err)

		return
	}

	h.writeJSON(w, r, http.StatusOK, dto.NewConnectionResponse(connection))
}

// readVerification decodes the body and checks the code. It writes the problem and
// answers false when the body is refused.
func readVerification(
	w http.ResponseWriter,
	r *http.Request,
	target *dto.VerificationRequest,
) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(target)

	var syntaxErr *json.SyntaxError

	switch {
	case err == nil:
	case !errors.As(err, &syntaxErr) && strings.HasPrefix(err.Error(), "json: unknown field "):
		problem.Write(w, r, problem.NewMemberUnknown().WithDetail(
			"The body carries "+strings.TrimPrefix(err.Error(), "json: unknown field ")+".",
		))

		return false
	default:
		problem.Write(w, r, problem.NewBodyInvalid())

		return false
	}

	if !isCode(target.Code) {
		problem.Write(w, r, problem.NewValidationFailed(problem.FieldFailure{
			Detail:  "must hold 4 to 8 digits",
			Pointer: "#/code",
		}))

		return false
	}

	return true
}

func isCode(code string) bool {
	if len(code) < minCodeLength || len(code) > maxCodeLength {
		return false
	}

	for _, char := range code {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// writeJSON answers the status with the body. The status line is gone when the encode
// fails, so the failure goes to the log alone.
func (h *ConnectionHandler) writeJSON(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	body any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to encode the answer", slog.Any("error", err))
	}
}
