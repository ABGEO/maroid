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
	pathTransactions     = "/transactions"
	pathTransactionTotal = "/transactions/total"
	pathChargeMonths     = "/charge-months"
)

// TransactionReader reads the stored transactions of the acting user.
type TransactionReader interface {
	TransactionsPage(
		ctx context.Context,
		asked page.Request,
		filter model.TransactionFilter,
	) (page.Window[model.Transaction], error)
	TransactionTotal(
		ctx context.Context,
		filter model.TransactionFilter,
	) (model.TransactionTotal, error)
	ChargeMonths(ctx context.Context, from *time.Time) ([]model.ChargeMonth, error)
}

// TransactionHandler serves the transactions page and the chart of the overview.
type TransactionHandler struct {
	logger       *slog.Logger
	transactions TransactionReader
}

// NewTransactionHandler creates a new TransactionHandler.
func NewTransactionHandler(
	logger *slog.Logger,
	transactions TransactionReader,
) *TransactionHandler {
	return &TransactionHandler{logger: loggerOf(logger, "transaction"), transactions: transactions}
}

// Routes returns the HTTP routes of the transactions.
func (h *TransactionHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		historyRoute(pathTransactions, h.List),
		historyRoute(pathTransactionTotal, h.Total),
		historyRoute(pathChargeMonths, h.ChargeMonths),
	}
}

// List handles GET /transactions.
func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(r, page.Options{
		Filters: []string{filterFrom, filterTo, filterKind},
	})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	filter, failure := readTransactionFilter(r.Context(), asked.Filters)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	window, err := h.transactions.TransactionsPage(r.Context(), asked, filter)
	if err != nil {
		writeFailure(w, r, h.logger, "listing the transactions", err)

		return
	}

	writePage(
		w, r, h.logger, dto.NewTransactionResponseList(window.Items), window.Next, window.Prev,
	)
}

// Total handles GET /transactions/total.
func (h *TransactionHandler) Total(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(r, page.Options{
		Bounded: true,
		Filters: []string{filterFrom, filterTo, filterKind},
	})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	filter, failure := readTransactionFilter(r.Context(), asked.Filters)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	total, err := h.transactions.TransactionTotal(r.Context(), filter)
	if err != nil {
		writeFailure(w, r, h.logger, "summing the transactions", err)

		return
	}

	writeOK(w, r, h.logger, dto.NewTransactionTotalResponse(total))
}

// ChargeMonths handles GET /charge-months.
func (h *TransactionHandler) ChargeMonths(w http.ResponseWriter, r *http.Request) {
	from, failure := readFrom(r)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	months, err := h.transactions.ChargeMonths(r.Context(), from)
	if err != nil {
		writeFailure(w, r, h.logger, "summing the months", err)

		return
	}

	writePage(w, r, h.logger, dto.NewChargeMonthResponseList(months), nil, nil)
}
