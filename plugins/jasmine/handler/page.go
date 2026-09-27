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

// listPage answers one page of a collection.
//
// It reads one row more than the page holds, and a row that remains proves that
// a further page exists. The read is a keyset, so the cost of a page does not
// grow with its position.
func listPage[T any, R any](
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	db *pluginapi.PluginDB,
	action string,
	read func(ctx context.Context, tx *sqlx.Tx, after string, limit int) ([]T, error),
	identify func(T) string,
	present func([]T) []R,
) {
	asked, failure := page.ReadRequest(r, page.Options{})
	if failure != nil {
		problem.Write(w, r, *failure)

		return
	}

	after := ""
	if asked.Cursor != nil {
		after = asked.Cursor.ID
	}

	rows, err := fetchInTx(
		r.Context(), db, action,
		func(ctx context.Context, tx *sqlx.Tx) ([]T, error) {
			return read(ctx, tx, after, asked.Limit+1)
		},
	)
	if err != nil {
		logger.ErrorContext(r.Context(), action, slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	var next *page.Cursor

	if len(rows) > asked.Limit {
		rows = rows[:asked.Limit]
		boundary := identify(rows[len(rows)-1])
		next = &page.Cursor{
			Sort:      asked.Sort,
			Direction: page.DirectionForward,
			Filters:   asked.Filters,
			Boundary:  map[string]string{"id": boundary},
			ID:        boundary,
		}
	}

	answered, err := page.New(r, present(rows), next, nil)
	if err != nil {
		logger.ErrorContext(r.Context(), action, slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, answered)
}
