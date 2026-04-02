package handler

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

// ErrHandlerAlreadyRegistered is returned when registering a handler for an ID that already has a handler registered.
var ErrHandlerAlreadyRegistered = errors.New("handler: already registered for id")

// Registry is a registry for handlers.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]Handler),
	}
}

// Register registers a handler for the given ID.
func (r *Registry) Register(id string, handler Handler) error {
	if _, exists := r.handlers[id]; exists {
		return fmt.Errorf("%w: %s", ErrHandlerAlreadyRegistered, id)
	}

	r.handlers[id] = handler

	return nil
}

// All returns all registered handlers.
func (r *Registry) All() []Handler {
	return slices.Collect(maps.Values(r.handlers))
}
