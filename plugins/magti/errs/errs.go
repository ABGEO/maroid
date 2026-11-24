// Package errs holds the sentinel errors of the plugin.
package errs

import "errors"

var (
	// ErrRequestFailed reports a call to the portal that failed or got an answer that is
	// not a success.
	ErrRequestFailed = errors.New("magti: request failed")
	// ErrSessionRejected reports a session that the portal no longer accepts.
	ErrSessionRejected = errors.New("magti: session rejected")
	// ErrNotConnected reports that Maroid holds no session for the acting workspace.
	ErrNotConnected = errors.New("magti: not connected")
	// ErrCredentialsAbsent reports a workspace that stored no username or no password.
	ErrCredentialsAbsent = errors.New("magti: credentials absent")
	// ErrCredentialsRejected reports a sign in that the portal rejected.
	ErrCredentialsRejected = errors.New("magti: credentials rejected")
	// ErrCodeRequired reports an account that asks for a code by phone at the sign in.
	ErrCodeRequired = errors.New("magti: the account asks for a code")
	// ErrAccountLocked reports an account that the portal locked.
	ErrAccountLocked = errors.New("magti: account locked")
	// ErrNotAssigned reports a workspace that holds no assigned subscriber.
	ErrNotAssigned = errors.New("magti: no assigned subscriber")
	// ErrHomeLine reports an assigned subscriber that is a home line, which holds no
	// traffic and no money entries.
	ErrHomeLine = errors.New("magti: the assigned subscriber is a home line")
	// ErrSubscriberUnknown reports a subscriber that the account does not hold.
	ErrSubscriberUnknown = errors.New("magti: subscriber unknown")
	// ErrSubscriberUnsupported reports a subscriber that is not a mobile subscriber or a
	// home line.
	ErrSubscriberUnsupported = errors.New("magti: subscriber unsupported")
	// ErrSubscriberAbsent reports an assigned subscriber that the account no longer holds.
	ErrSubscriberAbsent = errors.New("magti: assigned subscriber absent from the account")
	// ErrReversedPeriod reports a period whose first day is after its last day.
	ErrReversedPeriod = errors.New("magti: the first day is after the last day")
)
