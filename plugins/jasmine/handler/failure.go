package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// failWrite answers a write that did not land. A record that moved after the
// client read it answers 412. Every other cause answers 500, and the log record
// is the one report of it, because the body carries no cause.
func failWrite(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	message string,
	err error,
) {
	if errors.Is(err, precondition.ErrModified) {
		problem.Write(w, r, problem.NewPreconditionFailed())

		return
	}

	logger.ErrorContext(r.Context(), message, slog.Any("error", err))
	problem.Write(w, r, problem.NewInternal())
}
