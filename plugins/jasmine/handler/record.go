package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// readRecord answers one record with the validator that guards a write to it.
// A read that finds nothing answers 404, because a caller of this route names a
// record and a name that resolves to no record is the one failure it can make.
func readRecord[T any, R any](
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	db *pluginapi.PluginDB,
	action string,
	read func(ctx context.Context, tx *sqlx.Tx) (T, error),
	moment func(T) time.Time,
	present func(T) R,
) {
	row, err := fetchInTx(r.Context(), db, action, read)
	if err != nil {
		logger.ErrorContext(r.Context(), action, slog.Any("error", err))
		problem.Write(w, r, problem.NewNotFound())

		return
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(moment(row)))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, present(row))
}
