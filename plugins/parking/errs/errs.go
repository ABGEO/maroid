// Package errs holds the sentinel errors of the plugin.
package errs

import "errors"

var (
	// ErrRequestFailed reports a call to the parking service that failed or got a
	// status that is not 200.
	ErrRequestFailed = errors.New("parking: request failed")
	// ErrParkingPlaceNotFound reports a parking place that does not exist.
	ErrParkingPlaceNotFound = errors.New("parking: parking place not found")
	// ErrNotConnected reports that Maroid holds no session token for the acting user, or
	// that the parking service rejected the one it held.
	ErrNotConnected = errors.New("parking: not connected")
	// ErrCredentialsRejected reports a sign in that the parking service rejected.
	ErrCredentialsRejected = errors.New("parking: credentials rejected")
	// ErrVerificationUnsupported reports a sign in that asks for no code by phone.
	ErrVerificationUnsupported = errors.New("parking: verification unsupported")
	// ErrCodeRejected reports a code that the parking service rejected.
	ErrCodeRejected = errors.New("parking: code rejected")
	// ErrCredentialsAbsent reports a user who stored no username or no password.
	ErrCredentialsAbsent = errors.New("parking: credentials absent")
	// ErrPendingAbsent reports that no pending connection exists, or that it expired.
	ErrPendingAbsent = errors.New("parking: no pending connection")
	// ErrCodeWait reports a request of a new code before the wait ends.
	ErrCodeWait = errors.New("parking: the wait for a new code has not ended")
)
