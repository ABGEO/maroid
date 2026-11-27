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

const pathAssetValues = "/asset-values"

// AssetValueReader reads the asset values of the acting user.
type AssetValueReader interface {
	AssetValues(
		ctx context.Context,
		asked page.Request,
		from *time.Time,
	) (page.Window[model.AssetValue], error)
}

// AssetValueHandler serves the line of the overview chart.
type AssetValueHandler struct {
	logger *slog.Logger
	values AssetValueReader
}

// NewAssetValueHandler creates a new AssetValueHandler.
func NewAssetValueHandler(logger *slog.Logger, values AssetValueReader) *AssetValueHandler {
	return &AssetValueHandler{logger: loggerOf(logger, "asset_value"), values: values}
}

// Routes returns the HTTP route of the asset values.
func (h *AssetValueHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{{
		Method:     http.MethodGet,
		Pattern:    pathAssetValues,
		Handler:    h.List,
		Permission: permission.AccountRead,
	}}
}

// List handles GET /asset-values.
func (h *AssetValueHandler) List(w http.ResponseWriter, r *http.Request) {
	asked, from, failure := readFromPage(r)
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	window, err := h.values.AssetValues(r.Context(), asked, from)
	if err != nil {
		writeFailure(w, r, h.logger, "listing the asset values", err)

		return
	}

	writePage(w, r, h.logger, dto.NewAssetValueResponseList(window.Items), window.Next, window.Prev)
}
