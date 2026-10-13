// Package errs holds the sentinel errors of the plugin.
package errs

import "errors"

var (
	// ErrRequestFailed reports a call to the gas service that failed or got an answer
	// that is not a success.
	ErrRequestFailed = errors.New("tbilisi-energy: request failed")
	// ErrNotConnected reports that Maroid holds no token for the acting workspace, or
	// that the gas service rejected the one it held.
	ErrNotConnected = errors.New("tbilisi-energy: not connected")
	// ErrCredentialsAbsent reports a workspace that stored no username or no password.
	ErrCredentialsAbsent = errors.New("tbilisi-energy: credentials absent")
	// ErrCredentialsRejected reports a sign in that the gas service rejected.
	ErrCredentialsRejected = errors.New("tbilisi-energy: credentials rejected")
	// ErrSignInBlocked reports a sign in that the gas service holds for a human check.
	ErrSignInBlocked = errors.New("tbilisi-energy: sign in blocked")
	// ErrNotAssigned reports a workspace that holds no assigned customer.
	ErrNotAssigned = errors.New("tbilisi-energy: no assigned customer")
	// ErrCustomerUnknown reports a customer that the gas account does not hold.
	ErrCustomerUnknown = errors.New("tbilisi-energy: customer unknown")
	// ErrReversedPeriod reports a period whose first date is after its last date.
	ErrReversedPeriod = errors.New("tbilisi-energy: the first date is after the last date")
	// ErrCustomerAbsent reports an assigned customer that the gas account no longer holds.
	ErrCustomerAbsent = errors.New("tbilisi-energy: assigned customer absent from the account")
)
