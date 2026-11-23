package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// listPage answers one page of a collection. Every read of the page runs in one
// transaction, so the page and its links see one state of the tables.
func listPage[T any, R any](
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	db pluginapi.PluginDB,
	asked page.Request,
	read func(ctx context.Context, tx *sqlx.Tx, seek page.Seek) ([]T, error),
	identify func(T) string,
	present func(ctx context.Context, tx *sqlx.Tx, rows []T) ([]R, error),
) {
	var (
		window page.Window[T]
		items  []R
	)

	err := inTx(r.Context(), db, "reading the page", func(ctx context.Context, tx *sqlx.Tx) error {
		var txErr error

		fetch := func(ctx context.Context, seek page.Seek) ([]T, error) {
			return read(ctx, tx, seek)
		}

		if window, txErr = page.Read(ctx, asked, fetch, identify); txErr != nil {
			return fmt.Errorf("reading the rows: %w", txErr)
		}

		items, txErr = present(ctx, tx, window.Items)

		return txErr
	})
	if err != nil {
		failRead(w, r, logger, "failed to read the page", err)

		return
	}

	answered, err := page.New(r, items, window.Next, window.Prev)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to build the page", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, answered)
}
