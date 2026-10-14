package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/telmico/dto"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
	"github.com/abgeo/maroid/plugins/telmico/permission"
)

const (
	pathBills    = "/bills"
	pathDocument = "/bills/{billId}/document"
	segmentID    = "billId"
	segmentFile  = "/document"

	// canonicalUUIDLength is the length of a UUID in the form 8-4-4-4-12, the only form
	// that PostgreSQL takes.
	canonicalUUIDLength = 36
)

// BillLister reads one page of the bills of the assigned customer.
type BillLister interface {
	BillsPage(ctx context.Context, asked page.Request) (page.Window[model.Bill], error)
}

// DocumentOpener downloads the document of a stored bill.
type DocumentOpener interface {
	Open(ctx context.Context, id string) (model.Document, error)
}

// BillHandler serves the bills page: the bills of the assigned customer of the acting
// workspace, the latest month first, and the download of the document of each.
type BillHandler struct {
	logger    *slog.Logger
	bills     BillLister
	documents DocumentOpener
}

// NewBillHandler creates a new BillHandler.
func NewBillHandler(logger *slog.Logger, bills BillLister, documents DocumentOpener) *BillHandler {
	return &BillHandler{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "bill"),
		),
		bills:     bills,
		documents: documents,
	}
}

// Routes returns the HTTP routes of the bills.
func (h *BillHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathBills,
			Handler:    h.List,
			Permission: permission.AccountRead,
		},
		{
			Method:     http.MethodGet,
			Pattern:    pathDocument,
			Handler:    h.Document,
			Permission: permission.AccountRead,
		},
	}
}

// List handles GET /bills. Each bill names the absolute address of its document, below
// the address of the collection.
func (h *BillHandler) List(w http.ResponseWriter, r *http.Request) {
	asked, failure := page.ReadRequest(r, page.Options{})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	window, err := h.bills.BillsPage(r.Context(), asked)
	if errors.Is(err, errs.ErrNotAssigned) {
		writeNotAssigned(w, r)

		return
	}

	if err != nil {
		h.logger.ErrorContext(r.Context(), "listing the bills failed", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	collection := address.BaseFromContext(r.Context()) + r.URL.EscapedPath()
	documentOf := func(id string) string { return collection + "/" + id + segmentFile }

	writePage(
		w,
		r,
		h.logger,
		dto.NewBillResponseList(window.Items, documentOf),
		window.Next,
		window.Prev,
	)
}

// Document handles GET /bills/{billId}/document. A value that is no UUID names no bill,
// so it answers 404 before any query runs.
func (h *BillHandler) Document(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, segmentID)
	if _, err := uuid.Parse(id); err != nil || len(id) != canonicalUUIDLength {
		problem.Write(w, r, problem.NewNotFound())

		return
	}

	document, err := h.documents.Open(r.Context(), id)
	if err != nil {
		writeFailure(w, r, h.logger, "downloading a bill", err)

		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Length", strconv.Itoa(len(document.Content)))
	w.Header().
		Set("Content-Disposition", `attachment; filename="`+safeFilename(document.Filename)+`"`)
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)

	if _, err = w.Write(document.Content); err != nil {
		h.logger.WarnContext(r.Context(), "writing the document failed", slog.Any("error", err))
	}
}

// safeFilename keeps the letters, the digits, the hyphen, and the dot of a name that
// the portal gave, so the name cannot end the quoted value of the header.
func safeFilename(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '.':
			return r
		default:
			return -1
		}
	}, name)
}
