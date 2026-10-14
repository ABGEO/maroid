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
	"github.com/abgeo/maroid/plugins/telmico/dto"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
	"github.com/abgeo/maroid/plugins/telmico/permission"
)

const (
	pathTransactions = "/transactions"
	filterFrom       = "from"
)

// TransactionLister reads one page of the transactions of the assigned customer.
type TransactionLister interface {
	TransactionsPage(
		ctx context.Context,
		asked page.Request,
		from *time.Time,
	) (page.Window[model.Transaction], error)
}

// TransactionHandler serves the transactions page: the transactions of the assigned
// customer of the acting workspace, the latest date first.
type TransactionHandler struct {
	logger       *slog.Logger
	transactions TransactionLister
}

// NewTransactionHandler creates a new TransactionHandler.
func NewTransactionHandler(
	logger *slog.Logger,
	transactions TransactionLister,
) *TransactionHandler {
	return &TransactionHandler{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "transaction"),
		),
		transactions: transactions,
	}
}

// Routes returns the HTTP route of the transactions.
func (h *TransactionHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathTransactions,
			Handler:    h.List,
			Permission: permission.AccountRead,
		},
	}
}

// List handles GET /transactions.
func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
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

	window, err := h.transactions.TransactionsPage(r.Context(), asked, from)
	if errors.Is(err, errs.ErrNotAssigned) {
		writeNotAssigned(w, r)

		return
	}

	if err != nil {
		h.logger.ErrorContext(
			r.Context(),
			"listing the transactions failed",
			slog.Any("error", err),
		)
		problem.Write(w, r, problem.NewInternal())

		return
	}

	writePage(
		w,
		r,
		h.logger,
		dto.NewTransactionResponseList(window.Items),
		window.Next,
		window.Prev,
	)
}

// readFrom reads the filter from, a date YYYY-MM-DD of Tbilisi. An absent filter
// answers nil.
func readFrom(raw string) (*time.Time, *problem.Problem) {
	if raw == "" {
		return nil, nil
	}

	from, err := time.ParseInLocation(time.DateOnly, raw, model.Tbilisi())
	if err != nil {
		return nil, problem.NewRequestInvalid().
			WithDetail("The " + filterFrom + " parameter is not valid: it must be a date YYYY-MM-DD.")
	}

	return &from, nil
}

// writePage answers one page of a collection with its links.
func writePage[T any](
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	items []T,
	next *page.Cursor,
	prev *page.Cursor,
) {
	answered, err := page.New(r, items, next, prev)
	if err != nil {
		logger.ErrorContext(r.Context(), "building a page failed", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	writeOK(w, r, logger, answered)
}
