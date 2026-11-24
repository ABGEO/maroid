package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/magti/dto"
	"github.com/abgeo/maroid/plugins/magti/model"
	"github.com/abgeo/maroid/plugins/magti/permission"
	"github.com/abgeo/maroid/plugins/magti/service"
)

const pathState = "/subscriber-states/self"

// StateReader reads the subscriber state of the acting workspace from the portal.
type StateReader interface {
	Read(ctx context.Context) (model.State, error)
}

var _ StateReader = (*service.States)(nil)

// StateHandler serves the subscriber page.
type StateHandler struct {
	logger *slog.Logger
	states StateReader
}

// NewStateHandler creates a new StateHandler.
func NewStateHandler(logger *slog.Logger, states StateReader) *StateHandler {
	return &StateHandler{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "state"),
		),
		states: states,
	}
}

// Routes returns the HTTP routes of the subscriber page.
func (h *StateHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathState,
			Handler:    h.Read,
			Permission: permission.AccountRead,
		},
	}
}

// Read handles GET /subscriber-states/self.
func (h *StateHandler) Read(w http.ResponseWriter, r *http.Request) {
	state, err := h.states.Read(r.Context())
	if err != nil {
		writeFailure(w, r, h.logger, "reading the subscriber state", err)

		return
	}

	writeOK(w, r, h.logger, dto.NewStateResponse(state))
}
