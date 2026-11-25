package model

// ConnectionState names the state of the connection of a workspace.
type ConnectionState string

// The states of a connection.
const (
	ConnectionStateNotConnected ConnectionState = "not_connected"
	ConnectionStateConnected    ConnectionState = "connected"
)

// Connection is the connection of the acting workspace, with the vehicles that the car log
// service lists when the workspace is connected.
type Connection struct {
	State             ConnectionState
	CredentialsStored bool
	Vehicles          []Vehicle
}
