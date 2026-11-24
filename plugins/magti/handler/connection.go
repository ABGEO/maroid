package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/magti/dto"
	"github.com/abgeo/maroid/plugins/magti/model"
	"github.com/abgeo/maroid/plugins/magti/permission"
	"github.com/abgeo/maroid/plugins/magti/service"
)

const (
	pathConnection = "/connections/self"
	pathAssignment = "/subscriber-assignments/self"

	memberSubscriberNumber = "subscriber_number"
	maxAssignmentBody      = 4 << 10
	maxSubscriberNumberLen = 64
)

// ConnectionService connects the telecom account of the acting workspace, and assigns
// one of its subscribers to the workspace.
type ConnectionService interface {
	Status(ctx context.Context) (*model.Connection, error)
	Connect(ctx context.Context) (*model.Connection, error)
	Disconnect(ctx context.Context) error
	Assign(ctx context.Context, number string, ifMatch *time.Time) (*model.Connection, error)
}

var _ ConnectionService = (*service.Connections)(nil)

// ConnectionHandler serves the account page: it reads, starts, and ends the connection
// of the telecom account of the acting workspace, and assigns a subscriber.
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

// Routes returns the HTTP routes of the account page.
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

	h.writeConnection(w, r, connection)
}

// Connect handles PUT /connections/self. A repeat signs in again and leaves the
// workspace connected, so the method is PUT.
func (h *ConnectionHandler) Connect(w http.ResponseWriter, r *http.Request) {
	connection, err := h.connections.Connect(r.Context())
	if err != nil {
		writeFailure(w, r, h.logger, "connecting the telecom account", err)

		return
	}

	h.writeConnection(w, r, connection)
}

// Disconnect handles DELETE /connections/self.
func (h *ConnectionHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	if err := h.connections.Disconnect(r.Context()); err != nil {
		writeFailure(w, r, h.logger, "ending the connection", err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Assign handles PUT /subscriber-assignments/self.
func (h *ConnectionHandler) Assign(w http.ResponseWriter, r *http.Request) {
	var request dto.AssignmentRequest
	if !readAssignment(w, r, &request) {
		return
	}

	connection, err := h.connections.Assign(
		r.Context(),
		request.SubscriberNumber,
		precondition.IfMatchFromContext(r.Context()),
	)
	if err != nil {
		writeFailure(w, r, h.logger, "assigning the subscriber", err)

		return
	}

	h.writeConnection(w, r, connection)
}

// writeConnection answers the connection, with the validator of the assignment when the
// workspace holds one.
func (h *ConnectionHandler) writeConnection(
	w http.ResponseWriter,
	r *http.Request,
	connection *model.Connection,
) {
	if connection.Assignment != nil {
		w.Header().Set(precondition.ETagHeader, precondition.ETag(connection.Assignment.AssignedAt))
	}

	writeOK(w, r, h.logger, dto.NewConnectionResponse(connection))
}

// readAssignment decodes the body and checks the subscriber number. It writes the
// problem and answers false when the body is refused.
func readAssignment(w http.ResponseWriter, r *http.Request, target *dto.AssignmentRequest) bool {
	var members map[string]json.RawMessage

	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxAssignmentBody)).Decode(&members)

	var tooLarge *http.MaxBytesError

	switch {
	case errors.As(err, &tooLarge):
		problem.Write(
			w,
			r,
			problem.NewContentTooLarge().WithDetail("The body holds at most 4 KiB."),
		)

		return false
	case err != nil || members == nil:
		problem.Write(w, r, problem.NewBodyInvalid())

		return false
	}

	if unknown := unknownMembers(members, memberSubscriberNumber); len(unknown) > 0 {
		problem.Write(w, r, problem.NewMemberUnknown().WithDetail(
			"The body carries "+strings.Join(unknown, ", ")+".",
		))

		return false
	}

	raw, present := members[memberSubscriberNumber]
	if present && json.Unmarshal(raw, &target.SubscriberNumber) != nil {
		problem.Write(w, r, problem.NewBodyInvalid())

		return false
	}

	length := utf8.RuneCountInString(target.SubscriberNumber)
	if strings.TrimSpace(target.SubscriberNumber) == "" || length > maxSubscriberNumberLen {
		problem.Write(w, r, problem.NewValidationFailed(problem.FieldFailure{
			Detail:  "must hold 1 to 64 characters",
			Pointer: "#/" + memberSubscriberNumber,
		}))

		return false
	}

	return true
}

// unknownMembers answers the members of the body that the schema does not declare,
// each quoted, in a stable order.
func unknownMembers(members map[string]json.RawMessage, declared ...string) []string {
	var unknown []string

	for name := range members {
		if !slices.Contains(declared, name) {
			unknown = append(unknown, strconv.Quote(name))
		}
	}

	slices.Sort(unknown)

	return unknown
}
