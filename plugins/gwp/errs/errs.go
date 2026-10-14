// Package errs holds the sentinel errors of the plugin.
package errs

import "errors"

var (
	// ErrRequestFailed reports a call to the water service that failed or got a status
	// that is not a success.
	ErrRequestFailed = errors.New("gwp: request failed")
	// ErrNotConnected reports that Maroid holds no token for the acting workspace.
	ErrNotConnected = errors.New("gwp: not connected")
	// ErrTokenRejected reports a token that the water service no longer accepts.
	ErrTokenRejected = errors.New("gwp: token rejected")
	// ErrCredentialsAbsent reports a workspace that stored no username or no password.
	ErrCredentialsAbsent = errors.New("gwp: credentials absent")
	// ErrCredentialsRejected reports a sign in that the water service rejected.
	ErrCredentialsRejected = errors.New("gwp: credentials rejected")
	// ErrNotAssigned reports a workspace that holds no assigned customer.
	ErrNotAssigned = errors.New("gwp: no assigned customer")
	// ErrCustomerUnknown reports a customer that the water account does not hold.
	ErrCustomerUnknown = errors.New("gwp: customer unknown")
	// ErrCustomerAbsent reports readings of the water service that hold no entry for
	// the assigned customer.
	ErrCustomerAbsent = errors.New("gwp: assigned customer absent from the readings")
)
