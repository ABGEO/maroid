package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
// Every other failure answers 500, and the log record holds its cause.
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

	switch {
	case errors.Is(err, sql.ErrNoRows):
		problem.Write(w, r, problem.NewNotFound())

		return
	case err != nil:
		logger.ErrorContext(r.Context(), action, slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(moment(row)))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, present(row))
}

// replacer reads one record and writes its values back under a validator.
type replacer[T any] interface {
	GetByID(ctx context.Context, id string) (*T, error)
	Update(ctx context.Context, entity *T, ifMatch *time.Time) error
}

// replaceRecord applies the values of a request to the record and answers the
// stored record with its new validator. The read and the write run in one
// transaction, and failWrite answers a write that did not land.
func replaceRecord[T any, R any](
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	db *pluginapi.PluginDB,
	id string,
	open func(tx *sqlx.Tx) replacer[T],
	apply func(*T),
	moment func(*T) time.Time,
	present func(*T) R,
) {
	ifMatch := precondition.IfMatchFromContext(r.Context())

	record, err := fetchInTx(
		r.Context(),
		db,
		"replacing the record "+id,
		func(ctx context.Context, tx *sqlx.Tx) (*T, error) {
			repo := open(tx)

			existing, txErr := repo.GetByID(ctx, id)
			if txErr != nil {
				return nil, fmt.Errorf("getting the current record: %w", txErr)
			}

			apply(existing)

			if txErr = repo.Update(ctx, existing, ifMatch); txErr != nil {
				return nil, fmt.Errorf("writing the record: %w", txErr)
			}

			return existing, nil
		},
	)
	if err != nil {
		failWrite(w, r, logger, "failed to replace the record", err)

		return
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(moment(record)))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, present(record))
}
