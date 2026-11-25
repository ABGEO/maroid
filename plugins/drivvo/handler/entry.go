package handler

import (
	"net/http"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/drivvo/dto"
	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/repository"
)

const sortOccurredAt = "occurred_at"

// LogEntries handles GET /vehicles/{vehicleId}/log-entries.
func (h *CarLogHandler) LogEntries(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := vehicleOf(r)
	if err != nil {
		writeFailure(w, r, h.logger, "listing the log entries", err)

		return
	}

	asked, failure := page.ReadRequest(r, page.Options{
		SortFields: []string{sortOccurredAt},
		Filters:    []string{filterFrom, filterTo},
	})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	filter, failure := readFilter(r)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	entries, err := h.figures.LogEntries(
		r.Context(),
		vehicleID,
		filter,
		asked,
		orderOf(asked, repository.Descending),
	)
	if err != nil {
		writeFailure(w, r, h.logger, "listing the log entries", err)

		return
	}

	writePage(
		w, r, h.logger,
		dto.NewLogEntryItemResponseList(
			entries.Window.Items,
			entries.Consumption,
			entries.Settings.Currency,
		),
		entries.Window.Next,
		entries.Window.Prev,
	)
}

// LogEntryTotals handles GET /vehicles/{vehicleId}/log-entry-totals.
func (h *CarLogHandler) LogEntryTotals(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := vehicleOf(r)
	if err != nil {
		writeFailure(w, r, h.logger, "summing the log entries", err)

		return
	}

	kinds := r.URL.Query()[filterKind]
	if len(kinds) != 1 {
		problem.Write(w, r, refuseParameter(filterKind, "it is required once"))

		return
	}

	filter, failure := readFilter(r)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	totals, settings, err := h.figures.LogEntryTotals(r.Context(), vehicleID, filter)
	if err != nil {
		writeFailure(w, r, h.logger, "summing the log entries", err)

		return
	}

	writeOK(
		w,
		r,
		h.logger,
		dto.NewLogEntryTotalsResponse(filter.Kinds[0], totals, settings.Currency),
	)
}

// LogEntry handles GET /vehicles/{vehicleId}/log-entries/{logEntryId}.
func (h *CarLogHandler) LogEntry(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := vehicleOf(r)
	if err != nil {
		writeFailure(w, r, h.logger, "reading the log entry", err)

		return
	}

	entryID, err := pathID(r, paramLogEntryID, errs.ErrLogEntryNotFound)
	if err != nil {
		writeFailure(w, r, h.logger, "reading the log entry", err)

		return
	}

	entry, settings, err := h.figures.LogEntry(r.Context(), vehicleID, entryID)
	if err != nil {
		writeFailure(w, r, h.logger, "reading the log entry", err)

		return
	}

	writeOK(w, r, h.logger, dto.NewLogEntryResponse(entry, settings.Currency))
}

// readFilter reads the kinds and the period, with the bounds of the period in the zone of
// the member.
func readFilter(r *http.Request) (model.EntryFilter, *problem.Problem) {
	kinds, failure := readKinds(r)
	if failure != nil {
		return model.EntryFilter{}, failure
	}

	period, failure := readPeriod(r)
	if failure != nil {
		return model.EntryFilter{}, failure
	}

	start, end := period.Bounds(pluginapi.ActingTimeZone(r.Context()))

	return model.EntryFilter{Kinds: kinds, Start: start, End: end}, nil
}
