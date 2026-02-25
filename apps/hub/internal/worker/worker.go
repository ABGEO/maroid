// Package worker provides the Worker interface and background worker implementations.
package worker

import "context"

// Worker represents a background worker that can be started and stopped.
type Worker interface {
	// Name returns a stable, lowercase identifier for this worker type (e.g. "cron", "mqtt").
	Name() string
	// Prepare validates configuration and performs pre-start setup (e.g. registering jobs).
	// All workers are prepared before any worker is started.
	Prepare() error
	// Start begins the worker's main loop and blocks until ctx is cancelled.
	Start(ctx context.Context) error
	// Stop gracefully shuts down the worker within the provided context deadline.
	Stop(ctx context.Context) error
}
