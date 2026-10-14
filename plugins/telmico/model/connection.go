package model

// ConnectionState names whether Maroid holds a session for the electricity account of
// the acting workspace.
type ConnectionState string

const (
	// ConnectionStateNotConnected marks a workspace for which Maroid holds no session.
	ConnectionStateNotConnected ConnectionState = "not_connected"
	// ConnectionStateConnected marks a workspace whose credentials the portal accepted.
	ConnectionStateConnected ConnectionState = "connected"
)

// Connection is the state of the electricity account of the acting workspace.
type Connection struct {
	State             ConnectionState
	CredentialsStored bool
	// Customers is empty unless State is ConnectionStateConnected.
	Customers []Customer
}

// Customer is one customer of the electricity account, as the portal reports it.
type Customer struct {
	Number   string
	Name     string
	Address  string
	Assigned bool
}
