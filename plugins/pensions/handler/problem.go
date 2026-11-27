package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/pensions/errs"
)

//nolint:gosec // problem types that name a condition, not a credential.
const (
	typeCredentialsAbsent   = "/problems/dev.maroid.pensions/credentials-absent"
	typeCredentialsRejected = "/problems/dev.maroid.pensions/credentials-rejected"
	typeServiceUnavailable  = "/problems/dev.maroid.pensions/service-unavailable"
)

// writeFailure answers the failure of a call to a service of the plugin. Each answer
// names the condition, and none names a credential or the holder.
func writeFailure(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	action string,
	err error,
) {
	switch {
	case errors.Is(err, errs.ErrCredentialsAbsent):
		writeProblem(w, r, typeCredentialsAbsent, http.StatusUnprocessableEntity,
			"The settings hold no username or no password.",
			"Store the username and the password of the pension account in the settings.")
	case errors.Is(err, errs.ErrCredentialsRejected):
		writeProblem(w, r, typeCredentialsRejected, http.StatusUnprocessableEntity,
			"The pension service rejected the credentials.",
			"Check the username and the password in the settings.")
	case errors.Is(err, errs.ErrRequestFailed):
		logger.WarnContext(r.Context(), "the pension service failed",
			slog.String("action", action), slog.Any("error", err))
		writeProblem(w, r, typeServiceUnavailable, http.StatusBadGateway,
			"The pension service is unavailable.",
			"Try again later.")
	default:
		logger.ErrorContext(r.Context(), "a call of the plugin failed",
			slog.String("action", action), slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())
	}
}

func writeProblem(
	w http.ResponseWriter,
	r *http.Request,
	problemType string,
	status int,
	title, detail string,
) {
	problem.Write(w, r, &problem.Problem{
		Type:   problemType,
		Title:  title,
		Status: status,
		Detail: detail,
	})
}
