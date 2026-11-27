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
	pathPortfolioShares     = "/portfolio-shares"
	pathPortfolioUnitPrices = "/portfolio-unit-prices"
)

// PortfolioReader reads the portfolio shares of the acting user and their unit prices.
type PortfolioReader interface {
	PortfolioShares(ctx context.Context) ([]model.PortfolioShare, error)
	UnitPrices(
		ctx context.Context,
		asked page.Request,
		from *time.Time,
	) (page.Window[model.PortfolioUnitPrice], error)
}

// PortfolioHandler serves the portfolios page.
type PortfolioHandler struct {
	logger     *slog.Logger
	portfolios PortfolioReader
}

// NewPortfolioHandler creates a new PortfolioHandler.
func NewPortfolioHandler(logger *slog.Logger, portfolios PortfolioReader) *PortfolioHandler {
	return &PortfolioHandler{logger: loggerOf(logger, "portfolio"), portfolios: portfolios}
}

// Routes returns the HTTP routes of the portfolios.
func (h *PortfolioHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathPortfolioShares,
			Handler:    h.Shares,
			Permission: permission.AccountRead,
		},
		{
			Method:     http.MethodGet,
			Pattern:    pathPortfolioUnitPrices,
			Handler:    h.UnitPrices,
			Permission: permission.AccountRead,
		},
	}
}

// Shares handles GET /portfolio-shares.
func (h *PortfolioHandler) Shares(w http.ResponseWriter, r *http.Request) {
	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return
	}

	shares, err := h.portfolios.PortfolioShares(r.Context())
	if err != nil {
		writeFailure(w, r, h.logger, "listing the portfolio shares", err)

		return
	}

	writePage(w, r, h.logger, dto.NewPortfolioShareResponseList(shares), nil, nil)
}

// UnitPrices handles GET /portfolio-unit-prices.
func (h *PortfolioHandler) UnitPrices(w http.ResponseWriter, r *http.Request) {
	asked, from, failure := readFromPage(r)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	window, err := h.portfolios.UnitPrices(r.Context(), asked, from)
	if err != nil {
		writeFailure(w, r, h.logger, "listing the unit prices", err)

		return
	}

	writePage(w, r, h.logger, dto.NewUnitPriceResponseList(window.Items), window.Next, window.Prev)
}
