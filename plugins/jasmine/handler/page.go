package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// listPage answers one page of a collection. Every read of the page runs in one
// transaction, so the page and its links see one state of the table.
func listPage[T any, R any](
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	db *pluginapi.PluginDB,
	action string,
	read func(ctx context.Context, tx *sqlx.Tx, seek page.Seek) ([]T, error),
	identify func(T) string,
	present func([]T) []R,
) {
	asked, failure := page.ReadRequest(r, page.Options{})
	if failure != nil {
		problem.Write(w, r, *failure)

		return
	}

	window, err := fetchInTx(
		r.Context(), db, action,
		func(ctx context.Context, tx *sqlx.Tx) (page.Window[T], error) {
			fetch := func(ctx context.Context, seek page.Seek) ([]T, error) {
				return read(ctx, tx, seek)
			}

			return page.Read(ctx, asked, fetch, identify)
		},
	)
	if err != nil {
		logger.ErrorContext(r.Context(), action, slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	answered, err := page.New(r, present(window.Items), window.Next, window.Prev)
	if err != nil {
		logger.ErrorContext(r.Context(), action, slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, answered)
}
