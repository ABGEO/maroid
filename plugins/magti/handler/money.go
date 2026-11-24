package handler

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/magti/dto"
	"github.com/abgeo/maroid/plugins/magti/model"
	"github.com/abgeo/maroid/plugins/magti/permission"
	"github.com/abgeo/maroid/plugins/magti/repository"
)

const (
	pathMoneyEntries = "/money-entries"
	pathMonthTotals  = "/monthly-money-totals"
	sortOccurredAt   = "occurred_at"
	sortMonth        = "month"

	// maxMonthsBack bounds the first month of the totals, so they hold at most a year.
	maxMonthsBack = 11
)

// MoneyLister reads the money entries of the assigned subscriber, and their sums.
type MoneyLister interface {
	MoneyPage(
		ctx context.Context,
		asked page.Request,
		order repository.Order,
	) (page.Window[model.MoneyEntry], error)
	MonthTotals(ctx context.Context, from model.Month) ([]model.MonthTotal, error)
}

// MoneyHandler serves the money page.
type MoneyHandler struct {
	logger *slog.Logger
	money  MoneyLister
	now    func() time.Time
}

// NewMoneyHandler creates a new MoneyHandler. now gives the current month.
func NewMoneyHandler(logger *slog.Logger, money MoneyLister, now func() time.Time) *MoneyHandler {
	return &MoneyHandler{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "money"),
		),
		money: money,
		now:   now,
	}
}

// Routes returns the HTTP routes of the money entries and of their sums.
func (h *MoneyHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathMoneyEntries,
			Handler:    h.List,
			Permission: permission.AccountRead,
		},
		{
			Method:     http.MethodGet,
			Pattern:    pathMonthTotals,
			Handler:    h.Totals,
			Permission: permission.AccountRead,
		},
	}
}

// List handles GET /money-entries.
func (h *MoneyHandler) List(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(r, page.Options{SortFields: []string{sortOccurredAt}})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	window, err := h.money.MoneyPage(r.Context(), asked, orderOf(asked, repository.Descending))
	if err != nil {
		writeFailure(w, r, h.logger, "listing the money entries", err)

		return
	}

	writePage(w, r, h.logger, dto.NewMoneyEntryResponseList(window.Items), window.Next, window.Prev)
}

// Totals handles GET /monthly-money-totals. Every month from the first one to the current
// one answers, and a month with no entry answers zero.
func (h *MoneyHandler) Totals(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(r, page.Options{
		Bounded:    true,
		SortFields: []string{sortMonth},
		Filters:    []string{filterFrom},
	})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	current := model.MonthOf(h.now())

	from, failure := readFirstMonth(asked.Filters[filterFrom], current)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	sums, err := h.money.MonthTotals(r.Context(), from)
	if err != nil {
		writeFailure(w, r, h.logger, "summing the money entries", err)

		return
	}

	totals := fillMonths(sums, from, current)
	if orderOf(asked, repository.Descending) == repository.Descending {
		slices.Reverse(totals)
	}

	writePage(w, r, h.logger, dto.NewMonthTotalResponseList(totals), nil, nil)
}

// readFirstMonth reads the filter from, a month YYYY-MM, required, from 11 months before
// the current month to the current month.
func readFirstMonth(raw string, current model.Month) (model.Month, *problem.Problem) {
	if raw == "" {
		return model.Month{}, refuseParameter(filterFrom, "it is required")
	}

	from, err := model.ParseMonth(raw)
	if err != nil {
		return model.Month{}, refuseParameter(filterFrom, "it must be a month YYYY-MM")
	}

	if current.AddMonths(-maxMonthsBack).After(from) || from.After(current) {
		return model.Month{}, refuseParameter(
			filterFrom,
			"it must lie within the 11 months before the current month",
		)
	}

	return from, nil
}

// fillMonths answers each month from the first to the last, the earliest first, with
// the sums that the database holds and zero for the others.
func fillMonths(sums []model.MonthTotal, from, last model.Month) []model.MonthTotal {
	held := make(map[model.Month]model.MonthTotal, len(sums))
	for _, sum := range sums {
		held[sum.Month] = sum
	}

	var totals []model.MonthTotal

	for month := from; !month.After(last); month = month.Next() {
		total, found := held[month]
		if !found {
			total = model.MonthTotal{Month: month}
		}

		totals = append(totals, total)
	}

	return totals
}
