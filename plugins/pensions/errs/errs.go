// Package errs holds the sentinel errors of the plugin.
package errs

import "errors"

var (
	// ErrRequestFailed reports a call to the pension service that failed or got an answer
	// that is not a success.
	ErrRequestFailed = errors.New("pensions: request failed")
	// ErrSessionRejected reports a session that the pension service no longer accepts.
	ErrSessionRejected = errors.New("pensions: session rejected")
	// ErrNotConnected reports that Maroid holds no session for the acting user.
	ErrNotConnected = errors.New("pensions: not connected")
	// ErrCredentialsAbsent reports a user that stored no username or no password.
	ErrCredentialsAbsent = errors.New("pensions: credentials absent")
	// ErrCredentialsRejected reports a sign in that the pension service rejected.
	ErrCredentialsRejected = errors.New("pensions: credentials rejected")
	// ErrReversedPeriod reports a period whose first month is after its last month.
	ErrReversedPeriod = errors.New("pensions: the first month is after the last month")
	// ErrPeriodNotEnded reports a period whose last month has not ended for every user.
	ErrPeriodNotEnded = errors.New("pensions: the last month has not ended")
)
