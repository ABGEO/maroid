package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/drivvo/errs"
)

//nolint:gosec // problem types that name a condition, not a credential.
const (
	typeCredentialsAbsent   = "/problems/dev.maroid.drivvo/credentials-absent"
	typeCredentialsRejected = "/problems/dev.maroid.drivvo/credentials-rejected"
	typeNewPasswordRequired = "/problems/dev.maroid.drivvo/new-password-required"
	typeServiceUnavailable  = "/problems/dev.maroid.drivvo/service-unavailable"
)

const titleServiceUnavailable = "Drivvo is unavailable."

// failure is one condition that a route answers as a problem of the plugin.
type failure struct {
	err    error
	kind   string
	status int
	title  string
	detail string
}

// failures holds the condition of each sentinel error. A refusal comes before a failed
// request, because a refusal names what the editor must change.
//
//nolint:gochecknoglobals // a fixed table that no code changes.
var failures = []failure{
	{
		errs.ErrCredentialsAbsent, typeCredentialsAbsent, http.StatusUnprocessableEntity,
		"The settings hold no email address or no password.",
		"Store the email address and the password of the Drivvo account in the settings.",
	},
	{
		errs.ErrCredentialsRejected, typeCredentialsRejected, http.StatusUnprocessableEntity,
		"Drivvo rejected the credentials.",
		"Check the email address and the password in the settings. " +
			"An account that signs in with Google needs a Drivvo password.",
	},
	{
		errs.ErrNewPasswordRequired, typeNewPasswordRequired, http.StatusUnprocessableEntity,
		"The Drivvo account asks for a new password.",
		"Choose a new password at Drivvo, store it in the settings, then connect again.",
	},
}

// writeFailure answers the failure of a call to a service of the plugin. Each answer names
// the condition, and none names a credential.
func writeFailure(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	action string,
	err error,
) {
	for _, known := range failures {
		if errors.Is(err, known.err) {
			writeProblem(w, r, known.kind, known.status, known.title, known.detail)

			return
		}
	}

	if errors.Is(err, errs.ErrRequestFailed) {
		logger.WarnContext(r.Context(), "the car log service failed",
			slog.String("action", action), slog.Any("error", err))

		detail := "Try again later."
		if errors.Is(err, context.DeadlineExceeded) {
			detail = "Drivvo gave no answer in time. Try again later."
		}

		writeProblem(w, r, typeServiceUnavailable, http.StatusBadGateway,
			titleServiceUnavailable, detail)

		return
	}

	logger.ErrorContext(r.Context(), "a call of the plugin failed",
		slog.String("action", action), slog.Any("error", err))
	problem.Write(w, r, problem.NewInternal())
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

// writeOK answers 200 with the body. The status line is gone when the encode fails, so the
// failure goes to the log alone.
func writeOK(w http.ResponseWriter, r *http.Request, logger *slog.Logger, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		logger.ErrorContext(r.Context(), "failed to encode the answer", slog.Any("error", err))
	}
}
