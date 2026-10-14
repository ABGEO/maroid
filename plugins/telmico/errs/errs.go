// Package errs holds the sentinel errors of the plugin.
package errs

import "errors"

var (
	// ErrRequestFailed reports a call to the portal that failed or got an answer that is
	// not a success.
	ErrRequestFailed = errors.New("telmico: request failed")
	// ErrSessionRejected reports a session that the portal no longer accepts.
	ErrSessionRejected = errors.New("telmico: session rejected")
	// ErrNotConnected reports that Maroid holds no session for the acting workspace.
	ErrNotConnected = errors.New("telmico: not connected")
	// ErrCredentialsAbsent reports a workspace that stored no username or no password.
	ErrCredentialsAbsent = errors.New("telmico: credentials absent")
	// ErrCredentialsRejected reports a sign in that the portal rejected.
	ErrCredentialsRejected = errors.New("telmico: credentials rejected")
	// ErrNotAssigned reports a workspace that holds no assigned customer.
	ErrNotAssigned = errors.New("telmico: no assigned customer")
	// ErrCustomerUnknown reports a customer that the account does not hold.
	ErrCustomerUnknown = errors.New("telmico: customer unknown")
	// ErrCustomerAbsent reports an assigned customer that the account no longer holds.
	ErrCustomerAbsent = errors.New("telmico: assigned customer absent from the account")
	// ErrBillUnknown reports a bill that Maroid holds for no assigned customer of the
	// acting workspace.
	ErrBillUnknown = errors.New("telmico: bill unknown")
	// ErrReversedPeriod reports a period whose first month is after its last month.
	ErrReversedPeriod = errors.New("telmico: the first month is after the last month")
)
