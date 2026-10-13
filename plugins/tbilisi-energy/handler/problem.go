package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/errs"
)

//nolint:gosec // problem types that name a condition, not a credential.
const (
	typeCredentialsAbsent   = "/problems/dev.maroid.tbilisi-energy/credentials-absent"
	typeCredentialsRejected = "/problems/dev.maroid.tbilisi-energy/credentials-rejected"
	typeSignInBlocked       = "/problems/dev.maroid.tbilisi-energy/sign-in-blocked"
	typeNotConnected        = "/problems/dev.maroid.tbilisi-energy/not-connected"
	typeCustomerUnknown     = "/problems/dev.maroid.tbilisi-energy/customer-unknown"
	typeServiceUnavailable  = "/problems/dev.maroid.tbilisi-energy/service-unavailable"
	typeNotAssigned         = "/problems/dev.maroid.tbilisi-energy/not-assigned"
)

// writeFailure answers the failure of a call to the connection service. Each answer
// names the condition, and none names a credential.
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
			"Store the username and the password of the gas account in the settings.")
	case errors.Is(err, errs.ErrCredentialsRejected):
		writeProblem(w, r, typeCredentialsRejected, http.StatusUnprocessableEntity,
			"The gas service rejected the credentials.",
			"Check the username and the password in the settings.")
	case errors.Is(err, errs.ErrSignInBlocked):
		writeProblem(w, r, typeSignInBlocked, http.StatusUnprocessableEntity,
			"The gas service holds the sign in for a human check.",
			"Sign in once at my.te.ge, then connect again.")
	case errors.Is(err, errs.ErrNotConnected):
		writeProblem(w, r, typeNotConnected, http.StatusConflict,
			"The workspace holds no connection to the gas service.",
			"Connect the gas account first.")
	case errors.Is(err, errs.ErrCustomerUnknown):
		writeProblem(w, r, typeCustomerUnknown, http.StatusUnprocessableEntity,
			"The gas account holds no such customer.",
			"Assign a customer from the list of the connection.")
	case errors.Is(err, errs.ErrRequestFailed):
		logger.WarnContext(r.Context(), "the gas service failed",
			slog.String("action", action), slog.Any("error", err))
		writeProblem(w, r, typeServiceUnavailable, http.StatusBadGateway,
			"The gas service is unavailable.",
			"Try again later.")
	default:
		logger.ErrorContext(r.Context(), "a call to the connection failed",
			slog.String("action", action), slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())
	}
}

// writeNotAssigned answers a read that needs the assigned customer of a workspace
// that holds none.
func writeNotAssigned(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, r, typeNotAssigned, http.StatusConflict,
		"The workspace holds no assigned customer.",
		"Connect the gas account and assign a customer on the account page.")
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
