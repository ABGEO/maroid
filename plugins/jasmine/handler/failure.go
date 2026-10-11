package handler

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/jasmine/errs"
)

// failWrite answers a write that did not land. A write whose precondition failed
// answers 412, a write to a record that does not exist answers 404, and a plant
// that names an environment of no record answers 422. Every other cause answers
// 500, and the log record is the one report of it, because the body carries no
// cause.
func failWrite(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	message string,
	err error,
) {
	switch {
	case preconditionFailed(r, err):
		problem.Write(w, r, problem.NewPreconditionFailed())
	case errors.Is(err, sql.ErrNoRows):
		problem.Write(w, r, problem.NewNotFound())
	case errors.Is(err, errs.ErrEnvironmentUnknown):
		writeFieldProblem(w, r, "environment_id", "names no environment of the workspace")
	default:
		logger.ErrorContext(r.Context(), message, slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())
	}
}

// preconditionFailed reports whether the write failed its If-Match. RFC 9110
// fails If-Match when no current record exists, so a write that named a
// validator for an absent record fails it too.
func preconditionFailed(r *http.Request, err error) bool {
	if errors.Is(err, precondition.ErrModified) {
		return true
	}

	namedValidator := precondition.IfMatchFromContext(r.Context()) != nil

	return namedValidator && errors.Is(err, sql.ErrNoRows)
}
