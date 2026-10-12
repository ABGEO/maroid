package model

// ConnectionState names whether Maroid holds a working token for the water account
// of the acting workspace.
type ConnectionState string

const (
	// ConnectionStateNotConnected marks a workspace for which Maroid holds no token.
	ConnectionStateNotConnected ConnectionState = "not_connected"
	// ConnectionStateConnected marks a workspace whose token the water service accepted.
	ConnectionStateConnected ConnectionState = "connected"
)

// Connection is the state of the water account of the acting workspace.
type Connection struct {
	State             ConnectionState
	CredentialsStored bool
	// Customers is empty unless State is ConnectionStateConnected.
	Customers []Customer
}

// Customer is one customer of the water account, as the water service reports it.
type Customer struct {
	Number string
	// CommunalNumber is empty when the water service gives none.
	CommunalNumber string
	Address        string
	Assigned       bool
}
