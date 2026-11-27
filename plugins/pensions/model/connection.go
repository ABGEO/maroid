package model

// ConnectionState names whether Maroid holds a session for the pension account of the
// acting user.
type ConnectionState string

const (
	// ConnectionStateNotConnected marks a user for whom Maroid holds no session.
	ConnectionStateNotConnected ConnectionState = "not_connected"
	// ConnectionStateConnected marks a user whose credentials the pension service accepted.
	ConnectionStateConnected ConnectionState = "connected"
)

// Connection is the state of the pension account of the acting user.
type Connection struct {
	State             ConnectionState
	CredentialsStored bool
	// Holder is nil unless State is ConnectionStateConnected.
	Holder *Holder
}

// Holder is the person of a connected pension account, as the pension service reports it.
type Holder struct {
	FullName   string
	AssetValue float64
}
