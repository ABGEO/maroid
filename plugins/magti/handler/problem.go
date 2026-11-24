package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/magti/errs"
)

//nolint:gosec // problem types that name a condition, not a credential.
const (
	typeCredentialsAbsent     = "/problems/dev.maroid.magti/credentials-absent"
	typeCredentialsRejected   = "/problems/dev.maroid.magti/credentials-rejected"
	typeCodeRequired          = "/problems/dev.maroid.magti/code-required"
	typeAccountLocked         = "/problems/dev.maroid.magti/account-locked"
	typeNotConnected          = "/problems/dev.maroid.magti/not-connected"
	typeNotAssigned           = "/problems/dev.maroid.magti/not-assigned"
	typeHomeLine              = "/problems/dev.maroid.magti/home-line"
	typeSubscriberUnknown     = "/problems/dev.maroid.magti/subscriber-unknown"
	typeSubscriberUnsupported = "/problems/dev.maroid.magti/subscriber-unsupported"
	typeServiceUnavailable    = "/problems/dev.maroid.magti/service-unavailable"
)

const titleServiceUnavailable = "Magti is unavailable."

// failure is one condition that a route answers as a problem of the plugin.
type failure struct {
	err    error
	kind   string
	status int
	title  string
	detail string
}

// failures holds the condition of each sentinel error. The order matters: a refusal of
// the sign in wraps no connection, and the more precise error comes first.
//
//nolint:gochecknoglobals // a fixed table that no code changes.
var failures = []failure{
	{
		errs.ErrCredentialsAbsent, typeCredentialsAbsent, http.StatusUnprocessableEntity,
		"The settings hold no username or no password.",
		"Store the phone number and the password of the Magti account in the settings.",
	},
	{
		errs.ErrCredentialsRejected, typeCredentialsRejected, http.StatusUnprocessableEntity,
		"Magti rejected the credentials.",
		"Check the phone number and the password in the settings.",
	},
	{
		errs.ErrCodeRequired, typeCodeRequired, http.StatusUnprocessableEntity,
		"The Magti account asks for a code by phone at the sign in.",
		"Turn off the code at the sign in on the Magti portal, then connect again.",
	},
	{
		errs.ErrAccountLocked, typeAccountLocked, http.StatusUnprocessableEntity,
		"Magti locked the account.",
		"Unlock the account on the Magti portal, then connect again.",
	},
	{
		errs.ErrNotConnected, typeNotConnected, http.StatusConflict,
		"The workspace holds no connection to Magti.",
		"Connect the Magti account first.",
	},
	{
		errs.ErrNotAssigned, typeNotAssigned, http.StatusConflict,
		"The workspace holds no assigned subscriber.",
		"Connect the Magti account and assign a subscriber on the account page.",
	},
	{
		errs.ErrHomeLine, typeHomeLine, http.StatusConflict,
		"The assigned subscriber is a home line.",
		"A home line holds no traffic and no money entries.",
	},
	{
		errs.ErrSubscriberUnknown, typeSubscriberUnknown, http.StatusUnprocessableEntity,
		"The Magti account holds no such subscriber.",
		"Assign a subscriber from the list of the connection.",
	},
	{
		errs.ErrSubscriberUnsupported, typeSubscriberUnsupported, http.StatusUnprocessableEntity,
		"The subscriber is not a mobile number or a home line.",
		"Assign a mobile number or a home line.",
	},
}

// writeFailure answers the failure of a call to a service of the plugin. Each answer
// names the condition, and none names a credential.
func writeFailure(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	action string,
	err error,
) {
	if errors.Is(err, precondition.ErrModified) {
		problem.Write(w, r, problem.NewPreconditionFailed())

		return
	}

	for _, known := range failures {
		if errors.Is(err, known.err) {
			writeProblem(w, r, known.kind, known.status, known.title, known.detail)

			return
		}
	}

	if errors.Is(err, errs.ErrRequestFailed) {
		logger.WarnContext(r.Context(), "the portal failed",
			slog.String("action", action), slog.Any("error", err))

		detail := "Try again later."
		if errors.Is(err, context.DeadlineExceeded) {
			detail = "Magti gave no answer in time. Try again later."
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

// writeOK answers 200 with the body. The status line is gone when the encode fails,
// so the failure goes to the log alone.
func writeOK(w http.ResponseWriter, r *http.Request, logger *slog.Logger, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		logger.ErrorContext(r.Context(), "failed to encode the answer", slog.Any("error", err))
	}
}
