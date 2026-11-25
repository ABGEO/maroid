// Package errs holds the sentinel errors of the plugin.
package errs

import "errors"

var (
	// ErrRequestFailed reports a call to the car log service that failed, or got an answer
	// that is not a success or does not decode.
	ErrRequestFailed = errors.New("drivvo: request failed")
	// ErrSessionRejected reports a token that the car log service no longer accepts.
	ErrSessionRejected = errors.New("drivvo: session rejected")
	// ErrNotConnected reports that Maroid holds no session for the acting workspace.
	ErrNotConnected = errors.New("drivvo: not connected")
	// ErrCredentialsAbsent reports a workspace that stored no email address or no password.
	ErrCredentialsAbsent = errors.New("drivvo: credentials absent")
	// ErrCredentialsRejected reports a sign in that the car log service rejected.
	ErrCredentialsRejected = errors.New("drivvo: credentials rejected")
	// ErrNewPasswordRequired reports an account that asks for a new password at the sign in.
	ErrNewPasswordRequired = errors.New("drivvo: the account asks for a new password")
	// ErrVehicleNotFound reports a vehicle that the car log copy does not hold.
	ErrVehicleNotFound = errors.New("drivvo: vehicle not found")
	// ErrLogEntryNotFound reports a log entry that the car log copy does not hold for the
	// vehicle.
	ErrLogEntryNotFound = errors.New("drivvo: log entry not found")
	// ErrAmountUnreadable reports a stored amount that does not read as a number.
	ErrAmountUnreadable = errors.New("drivvo: an amount does not read")
	// ErrReversedPeriod reports a period whose first day is after its last day.
	ErrReversedPeriod = errors.New("drivvo: the first day is after the last day")
)
