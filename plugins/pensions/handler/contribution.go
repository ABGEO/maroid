package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/pensions/dto"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/permission"
)

const (
	pathContributions      = "/contributions"
	pathContributionTotal  = "/contributions/total"
	pathContributionMonths = "/contribution-months"
	pathEmployers          = "/employers"
)

// ContributionReader reads the contributions of the acting user.
type ContributionReader interface {
	ContributionsPage(
		ctx context.Context,
		asked page.Request,
		filter model.ContributionFilter,
	) (page.Window[model.Contribution], error)
	ContributionTotal(
		ctx context.Context,
		filter model.ContributionFilter,
	) (model.ContributionTotal, error)
	ContributionMonths(ctx context.Context, from *time.Time) ([]model.MonthSum, error)
	Employers(ctx context.Context) ([]model.Employer, error)
}

// ContributionHandler serves the contributions page and the columns of the overview chart.
type ContributionHandler struct {
	logger        *slog.Logger
	contributions ContributionReader
}

// NewContributionHandler creates a new ContributionHandler.
func NewContributionHandler(
	logger *slog.Logger,
	contributions ContributionReader,
) *ContributionHandler {
	return &ContributionHandler{
		logger:        loggerOf(logger, "contribution"),
		contributions: contributions,
	}
}

// Routes returns the HTTP routes of the contributions.
func (h *ContributionHandler) Routes() []pluginapi.Route {
	read := func(pattern string, handle http.HandlerFunc) pluginapi.Route {
		return pluginapi.Route{
			Method:     http.MethodGet,
			Pattern:    pattern,
			Handler:    handle,
			Permission: permission.AccountRead,
		}
	}

	return []pluginapi.Route{
		read(pathContributions, h.List),
		read(pathContributionTotal, h.Total),
		read(pathContributionMonths, h.Months),
		read(pathEmployers, h.Employers),
	}
}

// List handles GET /contributions.
func (h *ContributionHandler) List(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(r, page.Options{
		Filters: []string{filterEmployer, filterFrom, filterTo},
	})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	filter, failure := readContributionFilter(asked.Filters)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	window, err := h.contributions.ContributionsPage(r.Context(), asked, filter)
	if err != nil {
		writeFailure(w, r, h.logger, "listing the contributions", err)

		return
	}

	writePage(
		w,
		r,
		h.logger,
		dto.NewContributionResponseList(window.Items),
		window.Next,
		window.Prev,
	)
}

// Total handles GET /contributions/total.
func (h *ContributionHandler) Total(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(r, page.Options{
		Bounded: true,
		Filters: []string{filterEmployer, filterFrom, filterTo},
	})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	filter, failure := readContributionFilter(asked.Filters)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	total, err := h.contributions.ContributionTotal(r.Context(), filter)
	if err != nil {
		writeFailure(w, r, h.logger, "summing the contributions", err)

		return
	}

	writeOK(w, r, h.logger, dto.NewContributionTotalResponse(total))
}

// Months handles GET /contribution-months.
func (h *ContributionHandler) Months(w http.ResponseWriter, r *http.Request) {
	from, failure := readFrom(r)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	sums, err := h.contributions.ContributionMonths(r.Context(), from)
	if err != nil {
		writeFailure(w, r, h.logger, "summing the months", err)

		return
	}

	writePage(w, r, h.logger, dto.NewContributionMonthResponseList(sums), nil, nil)
}

// Employers handles GET /employers.
func (h *ContributionHandler) Employers(w http.ResponseWriter, r *http.Request) {
	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return
	}

	employers, err := h.contributions.Employers(r.Context())
	if err != nil {
		writeFailure(w, r, h.logger, "listing the employers", err)

		return
	}

	writePage(w, r, h.logger, dto.NewEmployerResponseList(employers), nil, nil)
}
