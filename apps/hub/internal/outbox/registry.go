package outbox

import (
	"errors"
	"fmt"
	"slices"
)

// ErrHandlerRegistered reports a second handler for a type.
var ErrHandlerRegistered = errors.New("outbox: a handler already serves the type")

// Registry holds one handler for each type.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

// Register stores the handlers, all of them or none. A type that the registry already
// holds, or that the handlers hold twice, refuses the whole call.
func (r *Registry) Register(handlers ...Handler) error {
	seen := make(map[string]struct{}, len(handlers))

	for _, handler := range handlers {
		taskType := handler.Type()

		_, held := r.handlers[taskType]
		_, repeated := seen[taskType]

		if held || repeated {
			return fmt.Errorf("%w: %s", ErrHandlerRegistered, taskType)
		}

		seen[taskType] = struct{}{}
	}

	for _, handler := range handlers {
		r.handlers[handler.Type()] = handler
	}

	return nil
}

// Get answers the handler of the type.
//
//nolint:ireturn // a type has a handler of any concrete kind.
func (r *Registry) Get(taskType string) (Handler, bool) {
	handler, found := r.handlers[taskType]

	return handler, found
}

// All answers every handler, in the order of its type.
func (r *Registry) All() []Handler {
	types := make([]string, 0, len(r.handlers))
	for taskType := range r.handlers {
		types = append(types, taskType)
	}

	slices.Sort(types)

	handlers := make([]Handler, 0, len(types))
	for _, taskType := range types {
		handlers = append(handlers, r.handlers[taskType])
	}

	return handlers
}
