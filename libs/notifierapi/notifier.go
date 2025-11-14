// Package notifierapi provides interfaces and types for a notification system
// that supports multiple transports and channels.
package notifierapi

import "context"

// Transport represents a generic notification transport.
type Transport interface {
	Send(ctx context.Context, msg Message) error
}

// Dispatcher defines an abstraction for sending notification messages
// to one or more logical channels. Each channel may include multiple
// transports (e.g., Telegram, Email, Webhook) for message delivery.
type Dispatcher interface {
	Send(ctx context.Context, channelName string, msg Message) error
	Channels() []string
}
