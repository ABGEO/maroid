package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/telmico/errs"
)

//nolint:gosec // problem types that name a condition, not a credential.
const (
	typeCredentialsAbsent   = "/problems/dev.maroid.telmico/credentials-absent"
	typeCredentialsRejected = "/problems/dev.maroid.telmico/credentials-rejected"
	typeNotConnected        = "/problems/dev.maroid.telmico/not-connected"
	typeCustomerUnknown     = "/problems/dev.maroid.telmico/customer-unknown"
	typeServiceUnavailable  = "/problems/dev.maroid.telmico/service-unavailable"
	typeNotAssigned         = "/problems/dev.maroid.telmico/not-assigned"
)

// writeFailure answers the failure of a call to a service of the plugin. Each answer
// names the condition, and none names a credential.
func writeFailure(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	action string,
	err error,
) {
	switch {
	case errors.Is(err, errs.ErrNotConnected):
		writeProblem(w, r, typeNotConnected, http.StatusConflict,
			"The workspace holds no connection to the electricity service.",
			"Connect the electricity account first.")
	case errors.Is(err, errs.ErrCredentialsAbsent):
		writeProblem(w, r, typeCredentialsAbsent, http.StatusUnprocessableEntity,
			"The settings hold no username or no password.",
			"Store the username and the password of the electricity account in the settings.")
	case errors.Is(err, errs.ErrCredentialsRejected):
		writeProblem(w, r, typeCredentialsRejected, http.StatusUnprocessableEntity,
			"The electricity service rejected the credentials.",
			"Check the username and the password in the settings.")
	case errors.Is(err, errs.ErrCustomerUnknown):
		writeProblem(w, r, typeCustomerUnknown, http.StatusUnprocessableEntity,
			"The electricity account holds no such customer.",
			"Assign a customer from the list of the connection.")
	case errors.Is(err, errs.ErrNotAssigned):
		writeNotAssigned(w, r)
	case errors.Is(err, errs.ErrBillUnknown):
		problem.Write(w, r, problem.NewNotFound())
	case errors.Is(err, errs.ErrRequestFailed):
		logger.WarnContext(r.Context(), "the electricity service failed",
			slog.String("action", action), slog.Any("error", err))
		writeProblem(w, r, typeServiceUnavailable, http.StatusBadGateway,
			"The electricity service is unavailable.",
			"Try again later.")
	default:
		logger.ErrorContext(r.Context(), "a call of the plugin failed",
			slog.String("action", action), slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())
	}
}

// writeNotAssigned answers a read that needs the assigned customer of a workspace
// that holds none.
func writeNotAssigned(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, r, typeNotAssigned, http.StatusConflict,
		"The workspace holds no assigned customer.",
		"Connect the electricity account and assign a customer on the account page.")
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
