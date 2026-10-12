package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/gwp/dto"
	"github.com/abgeo/maroid/plugins/gwp/errs"
	"github.com/abgeo/maroid/plugins/gwp/model"
	"github.com/abgeo/maroid/plugins/gwp/permission"
)

const (
	pathReadings = "/readings"
	filterFrom   = "from"
)

// ReadingLister reads one page of the readings of the assigned customer.
type ReadingLister interface {
	ReadingsPage(
		ctx context.Context,
		asked page.Request,
		from *time.Time,
	) (page.Window[model.Reading], error)
}

// ReadingHandler serves the readings page: the readings of the assigned customer of
// the acting workspace, the latest date first.
type ReadingHandler struct {
	logger   *slog.Logger
	readings ReadingLister
}

// NewReadingHandler creates a new ReadingHandler.
func NewReadingHandler(logger *slog.Logger, readings ReadingLister) *ReadingHandler {
	return &ReadingHandler{
		logger:   logger,
		readings: readings,
	}
}

// Routes returns the HTTP route of the readings.
func (h *ReadingHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathReadings,
			Handler:    h.List,
			Permission: permission.AccountRead,
		},
	}
}

// List handles GET /readings.
func (h *ReadingHandler) List(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(r, page.Options{Filters: []string{filterFrom}})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	from, failure := readFrom(asked.Filters[filterFrom])
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	window, err := h.readings.ReadingsPage(r.Context(), asked, from)
	if errors.Is(err, errs.ErrNotAssigned) {
		writeNotAssigned(w, r)

		return
	}

	if err != nil {
		h.logger.ErrorContext(r.Context(), "listing the readings failed", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	answered, err := page.New(
		r,
		dto.NewMeterReadingResponseList(window.Items),
		window.Next,
		window.Prev,
	)
	if err != nil {
		h.logger.ErrorContext(
			r.Context(),
			"building the page of the readings failed",
			slog.Any("error", err),
		)
		problem.Write(w, r, problem.NewInternal())

		return
	}

	writeOK(w, r, h.logger, answered)
}

// readFrom reads the filter from, a date YYYY-MM-DD. An absent filter answers nil.
func readFrom(raw string) (*time.Time, *problem.Problem) {
	if raw == "" {
		return nil, nil
	}

	from, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return nil, problem.NewRequestInvalid().
			WithDetail("The " + filterFrom + " parameter is not valid: it must be a date YYYY-MM-DD.")
	}

	return &from, nil
}
