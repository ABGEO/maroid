package handler

import (
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/service"
)

//nolint:gosec // problem types that name a condition, not a credential.
const (
	typeCredentialsAbsent       = "/problems/dev.maroid.parking/credentials-absent"
	typeCredentialsRejected     = "/problems/dev.maroid.parking/credentials-rejected"
	typeVerificationUnsupported = "/problems/dev.maroid.parking/verification-unsupported"
	typeCodeRejected            = "/problems/dev.maroid.parking/code-rejected"
	typeCodeWait                = "/problems/dev.maroid.parking/code-wait"
	typeServiceUnavailable      = "/problems/dev.maroid.parking/service-unavailable"
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
	var wait *service.CodeWaitError

	switch {
	case errors.Is(err, errs.ErrCredentialsAbsent):
		writeProblem(w, r, typeCredentialsAbsent, http.StatusUnprocessableEntity,
			"The settings hold no username or no password.",
			"Store the username and the password of the parking account in the settings.")
	case errors.Is(err, errs.ErrCredentialsRejected):
		writeProblem(w, r, typeCredentialsRejected, http.StatusUnprocessableEntity,
			"The parking service rejected the credentials.",
			"Check the username and the password in the settings.")
	case errors.Is(err, errs.ErrVerificationUnsupported):
		writeProblem(w, r, typeVerificationUnsupported, http.StatusUnprocessableEntity,
			"The parking account does not confirm a sign in by phone.",
			"Maroid connects a parking account that the parking service confirms with an SMS code.")
	case errors.Is(err, errs.ErrCodeRejected):
		writeProblem(w, r, typeCodeRejected, http.StatusUnprocessableEntity,
			"The parking service rejected the code.",
			"Type the code again, or ask for a new one.")
	case errors.As(err, &wait):
		seconds := strconv.Itoa(int(math.Ceil(wait.Left.Seconds())))
		w.Header().Set("Retry-After", seconds)
		writeProblem(w, r, typeCodeWait, http.StatusTooManyRequests,
			"The wait for a new code has not ended.",
			"Ask for a new code in "+seconds+" seconds.")
	case errors.Is(err, errs.ErrPendingAbsent):
		problem.Write(
			w,
			r,
			problem.NewNotFound().WithDetail("No pending connection exists. Connect again."),
		)
	case errors.Is(err, errs.ErrRequestFailed):
		logger.WarnContext(
			r.Context(),
			"the parking service failed",
			slog.String("action", action),
			slog.Any("error", err),
		)
		writeProblem(w, r, typeServiceUnavailable, http.StatusBadGateway,
			"The parking service is unavailable.",
			"Try again later.")
	default:
		logger.ErrorContext(
			r.Context(),
			"a call to the connection failed",
			slog.String("action", action),
			slog.Any("error", err),
		)
		problem.Write(w, r, problem.NewInternal())
	}
}

func writeProblem(
	w http.ResponseWriter,
	r *http.Request,
	problemType string,
	status int,
	title string,
	detail string,
) {
	problem.Write(w, r, &problem.Problem{
		Type:   problemType,
		Title:  title,
		Status: status,
		Detail: detail,
	})
}
