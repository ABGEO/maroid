package model

import "time"

// ConnectionState names whether Maroid holds a working session token for the
// parking account of the acting user.
type ConnectionState string

const (
	// ConnectionStateNotConnected marks a user for whom Maroid holds no token.
	ConnectionStateNotConnected ConnectionState = "not_connected"
	// ConnectionStateConnected marks a user whose token the parking service accepted.
	ConnectionStateConnected ConnectionState = "connected"
)

// Connection is the state of the parking account of the acting user.
type Connection struct {
	State             ConnectionState
	CredentialsStored bool
	// Account is nil unless State is ConnectionStateConnected.
	Account *Account
	// Pending is nil unless a pending connection exists.
	Pending *PendingConnection
}

// Account is what the parking service reports about the parking account.
type Account struct {
	FirstName        string
	LastName         string
	Balance          float64
	FreeParkingsLeft int
}

// PendingConnection is a sign in that waits for the code that the parking service sent.
type PendingConnection struct {
	// SignInID with a code gives a session token, so no answer of Maroid holds it.
	SignInID  string
	Phone     string
	ResendAt  time.Time
	ExpiresAt time.Time
}
