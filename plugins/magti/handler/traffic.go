package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/magti/dto"
	"github.com/abgeo/maroid/plugins/magti/model"
	"github.com/abgeo/maroid/plugins/magti/permission"
	"github.com/abgeo/maroid/plugins/magti/repository"
)

const (
	pathTraffic = "/traffic-days"
	filterFrom  = "from"
	filterTo    = "to"
	sortDay     = "day"

	// maxTrafficDays bounds the period of one read: a year, with its leap day.
	maxTrafficDays = 366
)

// TrafficLister reads one page of the traffic of the assigned subscriber.
type TrafficLister interface {
	TrafficPage(
		ctx context.Context,
		asked page.Request,
		period model.Period,
		order repository.Order,
	) (page.Window[model.TrafficDay], error)
}

// TrafficHandler serves the traffic page.
type TrafficHandler struct {
	logger  *slog.Logger
	traffic TrafficLister
}

// NewTrafficHandler creates a new TrafficHandler.
func NewTrafficHandler(logger *slog.Logger, traffic TrafficLister) *TrafficHandler {
	return &TrafficHandler{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "traffic"),
		),
		traffic: traffic,
	}
}

// Routes returns the HTTP route of the traffic.
func (h *TrafficHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathTraffic,
			Handler:    h.List,
			Permission: permission.AccountRead,
		},
	}
}

// List handles GET /traffic-days.
func (h *TrafficHandler) List(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(r, page.Options{
		SortFields: []string{sortDay},
		Filters:    []string{filterFrom, filterTo},
	})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	period, failure := readPeriod(asked.Filters)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	window, err := h.traffic.TrafficPage(
		r.Context(),
		asked,
		period,
		orderOf(asked, repository.Ascending),
	)
	if err != nil {
		writeFailure(w, r, h.logger, "listing the traffic", err)

		return
	}

	writePage(w, r, h.logger, dto.NewTrafficDayResponseList(window.Items), window.Next, window.Prev)
}

// readPeriod reads the filters from and to, two days YYYY-MM-DD of Tbilisi, both
// required, in order, and at most a year apart.
func readPeriod(filters map[string]string) (model.Period, *problem.Problem) {
	var days [2]model.Day

	for index, name := range []string{filterFrom, filterTo} {
		raw, present := filters[name]
		if !present {
			return model.Period{}, refuseParameter(name, "it is required")
		}

		day, err := model.ParseDay(raw)
		if err != nil {
			return model.Period{}, refuseParameter(name, "it must be a date YYYY-MM-DD")
		}

		days[index] = day
	}

	period := model.Period{From: days[0], To: days[1]}

	switch {
	case period.From.After(period.To):
		return model.Period{}, refuseParameter(filterTo, "it comes before from")
	case period.To.Sub(period.From) >= maxTrafficDays:
		return model.Period{}, refuseParameter(filterTo, "the period holds at most 366 days")
	}

	return period, nil
}
