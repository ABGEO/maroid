package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/permission"
	"github.com/abgeo/maroid/plugins/pulse/settings"
)

const (
	pathSettings = "/settings"

	pauseLimit = 366 * 24 * time.Hour
)

// SettingsHandler reads and writes the settings of the acting user.
type SettingsHandler struct {
	logger   *slog.Logger
	settings *settings.Service
	now      func() time.Time
}

// NewSettingsHandler creates a SettingsHandler.
func NewSettingsHandler(
	logger *slog.Logger,
	service *settings.Service,
	now func() time.Time,
) *SettingsHandler {
	return &SettingsHandler{logger: logger, settings: service, now: now}
}

// Routes returns the routes of the settings.
func (h *SettingsHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathSettings,
			Handler:    h.Get,
			Permission: permission.CheckinsUse,
		},
		{
			Method:     http.MethodPut,
			Pattern:    pathSettings,
			Handler:    h.Put,
			Permission: permission.CheckinsUse,
		},
	}
}

// settingsBody is the body of the settings, on the wire.
type settingsBody struct {
	Enabled     bool       `json:"enabled"`
	PausedUntil *time.Time `json:"paused_until"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func newSettingsBody(row *model.Settings) settingsBody {
	return settingsBody{
		Enabled:     row.Enabled,
		PausedUntil: row.PausedUntil,
		UpdatedAt:   row.UpdatedAt,
	}
}

// Get handles GET /settings.
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	current, err := h.settings.Get(r.Context())
	if err != nil {
		failRead(w, r, h.logger, "failed to read the settings", err)

		return
	}

	answer(w, r, http.StatusOK, current.UpdatedAt, newSettingsBody(current))
}

// settingsRequest is the body of PUT /settings. Both members are required, and
// paused_until takes null.
type settingsRequest struct {
	Enabled     *bool           `json:"enabled"`
	PausedUntil json.RawMessage `json:"paused_until"`

	now   time.Time
	until *time.Time
}

func (s *settingsRequest) validate() []problem.FieldFailure {
	var failures fieldFailures

	if s.Enabled == nil {
		failures.add("enabled", "is required")
	}

	switch {
	case s.PausedUntil == nil:
		failures.add("paused_until", "is required")
	case bytes.Equal(s.PausedUntil, []byte("null")):
	default:
		var until time.Time
		if err := json.Unmarshal(s.PausedUntil, &until); err != nil {
			failures.add("paused_until", "must be a date-time or null")

			break
		}

		// A pause ends in the future, at most 366 days ahead.
		if !until.After(s.now) || until.After(s.now.Add(pauseLimit)) {
			failures.add("paused_until", "must lie in the future, at most 366 days ahead")
		}

		s.until = &until
	}

	return failures
}

// Put handles PUT /settings. The validator "0" creates the row.
func (h *SettingsHandler) Put(w http.ResponseWriter, r *http.Request) {
	req := settingsRequest{now: h.now()}
	if !readBody(w, r, &req) {
		return
	}

	updated, err := h.settings.Put(
		r.Context(),
		*req.Enabled,
		req.until,
		precondition.IfMatchFromContext(r.Context()),
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to write the settings", err)

		return
	}

	answer(w, r, http.StatusOK, updated.UpdatedAt, newSettingsBody(updated))
}
