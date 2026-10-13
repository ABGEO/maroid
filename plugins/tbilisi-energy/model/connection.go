package model

// ConnectionState names whether Maroid holds a working token for the gas account of
// the acting workspace.
type ConnectionState string

const (
	// ConnectionStateNotConnected marks a workspace for which Maroid holds no token.
	ConnectionStateNotConnected ConnectionState = "not_connected"
	// ConnectionStateConnected marks a workspace whose token the gas service accepted.
	ConnectionStateConnected ConnectionState = "connected"
)

// Connection is the state of the gas account of the acting workspace.
type Connection struct {
	State             ConnectionState
	CredentialsStored bool
	// Customers is empty unless State is ConnectionStateConnected.
	Customers []Customer
}

// Customer is one customer of the gas account, as the gas service reports it.
type Customer struct {
	Number   string
	Name     string
	Address  string
	Assigned bool
}
