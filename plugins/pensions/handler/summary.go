package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/dto"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/permission"
)

const pathSummary = "/account-summaries/self"

// SummaryReader reads the stored summary of the acting user and whether they are connected.
type SummaryReader interface {
	Summary(ctx context.Context) (model.SummaryView, error)
}

// SummaryHandler serves the summary that every page reads first.
type SummaryHandler struct {
	logger  *slog.Logger
	summary SummaryReader
}

// NewSummaryHandler creates a new SummaryHandler.
func NewSummaryHandler(logger *slog.Logger, summary SummaryReader) *SummaryHandler {
	return &SummaryHandler{logger: loggerOf(logger, "summary"), summary: summary}
}

// Routes returns the HTTP route of the summary.
func (h *SummaryHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{{
		Method:     http.MethodGet,
		Pattern:    pathSummary,
		Handler:    h.Read,
		Permission: permission.AccountRead,
	}}
}

// Read handles GET /account-summaries/self.
func (h *SummaryHandler) Read(w http.ResponseWriter, r *http.Request) {
	view, err := h.summary.Summary(r.Context())
	if err != nil {
		writeFailure(w, r, h.logger, "reading the summary", err)

		return
	}

	writeOK(w, r, h.logger, dto.NewSummaryViewResponse(view))
}
