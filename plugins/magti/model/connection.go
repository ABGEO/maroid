package model

import "time"

// ConnectionState names whether Maroid holds a session for the telecom account of the
// acting workspace.
type ConnectionState string

const (
	// ConnectionStateNotConnected marks a workspace for which Maroid holds no session.
	ConnectionStateNotConnected ConnectionState = "not_connected"
	// ConnectionStateConnected marks a workspace whose credentials the portal accepted.
	ConnectionStateConnected ConnectionState = "connected"
)

// SubscriberKind names the kind of a subscriber of the telecom account.
type SubscriberKind string

const (
	// SubscriberKindMobile marks a mobile phone number.
	SubscriberKindMobile SubscriberKind = "mobile"
	// SubscriberKindHomeLine marks the internet and the television of a house.
	SubscriberKindHomeLine SubscriberKind = "home_line"
	// SubscriberKindOther marks a subscriber that the plugin does not follow, a domain
	// for example.
	SubscriberKindOther SubscriberKind = "other"
)

// Assignable reports whether a workspace can follow a subscriber of the kind.
func (k SubscriberKind) Assignable() bool {
	return k == SubscriberKindMobile || k == SubscriberKindHomeLine
}

// Connection is the state of the telecom account of the acting workspace.
type Connection struct {
	State             ConnectionState
	CredentialsStored bool
	// Subscribers is empty unless State is ConnectionStateConnected.
	Subscribers []Subscriber
	// Assignment is nil when the workspace holds no assigned subscriber.
	Assignment *Assignment
}

// Subscriber is one subscriber of the telecom account, as the portal reports it.
type Subscriber struct {
	Number   string
	Label    string
	Kind     SubscriberKind
	Assigned bool
}

// Assignment is the subscriber that the acting workspace follows.
type Assignment struct {
	SubscriberNumber string         `db:"subscriber_number"`
	Kind             SubscriberKind `db:"kind"`
	AssignedAt       time.Time      `db:"assigned_at"`
}

// IsHomeLine reports whether the assigned subscriber is a home line.
func (a Assignment) IsHomeLine() bool {
	return a.Kind == SubscriberKindHomeLine
}
