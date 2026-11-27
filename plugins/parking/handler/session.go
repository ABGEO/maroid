package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/parking/dto"
	"github.com/abgeo/maroid/plugins/parking/model"
)

const (
	pathSessions     = "/sessions"
	pathSessionTotal = "/sessions/total"
	pathPlates       = "/plates"
	pathLotUsages    = "/lot-usages"
)

// SessionReader reads the stored sessions of the acting user.
type SessionReader interface {
	SessionsPage(
		ctx context.Context,
		asked page.Request,
		filter model.SessionFilter,
	) (page.Window[model.Session], error)
	SessionTotal(ctx context.Context, filter model.SessionFilter) (model.SessionTotal, error)
	Plates(ctx context.Context) ([]string, error)
	LotUsages(ctx context.Context, from *time.Time) ([]model.LotUsage, error)
}

// SessionHandler serves the sessions page and the parking lots of the overview.
type SessionHandler struct {
	logger   *slog.Logger
	sessions SessionReader
}

// NewSessionHandler creates a new SessionHandler.
func NewSessionHandler(logger *slog.Logger, sessions SessionReader) *SessionHandler {
	return &SessionHandler{logger: loggerOf(logger, "session"), sessions: sessions}
}

// Routes returns the HTTP routes of the sessions.
func (h *SessionHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		historyRoute(pathSessions, h.List),
		historyRoute(pathSessionTotal, h.Total),
		historyRoute(pathPlates, h.Plates),
		historyRoute(pathLotUsages, h.LotUsages),
	}
}

// List handles GET /sessions.
func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(r, page.Options{
		Filters: []string{filterFrom, filterTo, filterPlate, filterFree},
	})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	filter, failure := readSessionFilter(r.Context(), asked.Filters)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	window, err := h.sessions.SessionsPage(r.Context(), asked, filter)
	if err != nil {
		writeFailure(w, r, h.logger, "listing the sessions", err)

		return
	}

	writePage(w, r, h.logger, dto.NewSessionResponseList(window.Items), window.Next, window.Prev)
}

// Total handles GET /sessions/total.
func (h *SessionHandler) Total(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(r, page.Options{
		Bounded: true,
		Filters: []string{filterFrom, filterTo, filterPlate, filterFree},
	})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	filter, failure := readSessionFilter(r.Context(), asked.Filters)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	total, err := h.sessions.SessionTotal(r.Context(), filter)
	if err != nil {
		writeFailure(w, r, h.logger, "summing the sessions", err)

		return
	}

	writeOK(w, r, h.logger, dto.NewSessionTotalResponse(total))
}

// Plates handles GET /plates.
func (h *SessionHandler) Plates(w http.ResponseWriter, r *http.Request) {
	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return
	}

	plates, err := h.sessions.Plates(r.Context())
	if err != nil {
		writeFailure(w, r, h.logger, "listing the plates", err)

		return
	}

	writePage(w, r, h.logger, dto.NewPlateResponseList(plates), nil, nil)
}

// LotUsages handles GET /lot-usages.
func (h *SessionHandler) LotUsages(w http.ResponseWriter, r *http.Request) {
	from, failure := readFrom(r)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	usages, err := h.sessions.LotUsages(r.Context(), from)
	if err != nil {
		writeFailure(w, r, h.logger, "counting the parking lots", err)

		return
	}

	writePage(w, r, h.logger, dto.NewLotUsageResponseList(usages), nil, nil)
}
