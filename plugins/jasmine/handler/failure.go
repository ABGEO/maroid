package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/rest"
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
	if errors.Is(err, rest.ErrModified) {
		rest.Write(w, r, rest.NewPreconditionFailed())

		return
	}

	logger.ErrorContext(r.Context(), message, slog.Any("error", err))
	rest.Write(w, r, rest.NewInternal())
}
