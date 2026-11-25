package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/drivvo/dto"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/permission"
	"github.com/abgeo/maroid/plugins/drivvo/repository"
	"github.com/abgeo/maroid/plugins/drivvo/service"
)

const (
	pathVehicles       = "/vehicles"
	pathOverview       = "/vehicles/{vehicleId}/overview"
	pathLogEntries     = "/vehicles/{vehicleId}/log-entries"
	pathLogEntryTotals = "/vehicles/{vehicleId}/log-entry-totals"
	pathLogEntry       = "/vehicles/{vehicleId}/log-entries/{logEntryId}"
	pathReminders      = "/vehicles/{vehicleId}/reminders"
)

// FiguresService answers the pages from the car log copy of the acting workspace.
type FiguresService interface {
	Vehicles(ctx context.Context) (service.VehicleList, error)
	Overview(
		ctx context.Context,
		vehicleID string,
		period model.Period,
	) (service.VehicleOverview, error)
	LogEntries(
		ctx context.Context,
		vehicleID string,
		filter model.EntryFilter,
		asked page.Request,
		order repository.Order,
	) (service.EntryPage, error)
	LogEntryTotals(
		ctx context.Context,
		vehicleID string,
		filter model.EntryFilter,
	) (model.EntryTotals, model.CarLogSettings, error)
	LogEntry(
		ctx context.Context,
		vehicleID, entryID string,
	) (model.EntryRow, model.CarLogSettings, error)
	Reminders(ctx context.Context, vehicleID string) (service.VehicleReminders, error)
}

var _ FiguresService = (*service.Figures)(nil)

// CarLogHandler serves the pages of the car log copy: the vehicles, the overview, the
// history and the lists, the detail of a log entry, and the reminders.
type CarLogHandler struct {
	logger  *slog.Logger
	figures FiguresService
}

// NewCarLogHandler creates a new CarLogHandler.
func NewCarLogHandler(logger *slog.Logger, figures FiguresService) *CarLogHandler {
	return &CarLogHandler{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "car_log"),
		),
		figures: figures,
	}
}

// Routes returns the HTTP routes of the pages.
func (h *CarLogHandler) Routes() []pluginapi.Route {
	read := func(pattern string, handle http.HandlerFunc) pluginapi.Route {
		return pluginapi.Route{
			Method:     http.MethodGet,
			Pattern:    pattern,
			Handler:    handle,
			Permission: permission.AccountRead,
		}
	}

	return []pluginapi.Route{
		read(pathVehicles, h.Vehicles),
		read(pathOverview, h.Overview),
		read(pathLogEntries, h.LogEntries),
		read(pathLogEntryTotals, h.LogEntryTotals),
		read(pathLogEntry, h.LogEntry),
		read(pathReminders, h.Reminders),
	}
}

// Vehicles handles GET /vehicles. The collection is bounded, so it answers one page.
func (h *CarLogHandler) Vehicles(w http.ResponseWriter, r *http.Request) {
	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return
	}

	list, err := h.figures.Vehicles(r.Context())
	if err != nil {
		writeFailure(w, r, h.logger, "listing the vehicles", err)

		return
	}

	writePage(
		w, r, h.logger,
		dto.NewVehicleResponseList(list.Vehicles, list.Default, list.Settings),
		nil, nil,
	)
}

// Overview handles GET /vehicles/{vehicleId}/overview.
func (h *CarLogHandler) Overview(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := vehicleOf(r)
	if err != nil {
		writeFailure(w, r, h.logger, "reading the overview", err)

		return
	}

	period, failure := readPeriod(r)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	overview, err := h.figures.Overview(r.Context(), vehicleID, period)
	if err != nil {
		writeFailure(w, r, h.logger, "reading the overview", err)

		return
	}

	writeOK(w, r, h.logger, dto.NewOverviewResponse(
		overview.Overview,
		overview.Period,
		overview.Settings.Currency,
	))
}

// Reminders handles GET /vehicles/{vehicleId}/reminders. The collection is bounded, so it
// answers one page.
func (h *CarLogHandler) Reminders(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := vehicleOf(r)
	if err != nil {
		writeFailure(w, r, h.logger, "listing the reminders", err)

		return
	}

	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return
	}

	reminders, err := h.figures.Reminders(r.Context(), vehicleID)
	if err != nil {
		writeFailure(w, r, h.logger, "listing the reminders", err)

		return
	}

	writePage(w, r, h.logger, dto.NewReminderResponseList(reminders.Reminders), nil, nil)
}
