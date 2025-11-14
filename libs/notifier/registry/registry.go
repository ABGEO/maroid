// Package registry provides a registry and factory system for notifier
// transports. It allows dynamic registration of notifier factories
// indexed by URL schemes and provides a safe, concurrent way to create
// Transport instances from URL strings.
package registry

import (
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"sync"

	"github.com/abgeo/maroid/libs/notifierapi"
)

var (
	// ErrNilFactory is returned when attempting to register a nil factory.
	ErrNilFactory = errors.New("factory is nil")
	// ErrSchemeAlreadyRegistered is returned when attempting to register
	// a factory for a scheme that is already registered.
	ErrSchemeAlreadyRegistered = errors.New("factory scheme is already registered")
	// ErrUnknownScheme is returned when attempting to create a notifier
	// from a URL with an unregistered scheme.
	ErrUnknownScheme = errors.New("unsupported notifier scheme")
)

// FactoryFunc is a function that creates a Transport from a URL.
type FactoryFunc func(u *url.URL) (notifierapi.Transport, error)

// Registry manages notifier factories indexed by URL scheme.
type Registry interface {
	Register(scheme string, factory FactoryFunc) error
	New(rawURL string) (notifierapi.Transport, error)
	Schemes() []string
}

// SchemeRegistry manages the registration and creation of notifiers
// based on URL schemes. It is safe for concurrent use.
type SchemeRegistry struct {
	mu        sync.RWMutex
	factories map[string]FactoryFunc
}

var _ Registry = (*SchemeRegistry)(nil)

// New creates a new empty SchemeRegistry.
func New() *SchemeRegistry {
	return &SchemeRegistry{
		factories: make(map[string]FactoryFunc),
	}
}

// Register adds a factory for the given URL scheme. It returns an error
// if the factory is nil or if the scheme is already registered.
func (r *SchemeRegistry) Register(scheme string, factory FactoryFunc) error {
	if factory == nil {
		return ErrNilFactory
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[scheme]; exists {
		return fmt.Errorf("%w: %s", ErrSchemeAlreadyRegistered, scheme)
	}

	r.factories[scheme] = factory

	return nil
}

// New parses the raw URL string and creates a Transport using the
// registered factory for the URL's scheme. It returns an error if the
// URL is invalid or if no factory is registered for the scheme.
func (r *SchemeRegistry) New(rawURL string) (notifierapi.Transport, error) {
	notifierURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid notifier URL: %w", err)
	}

	r.mu.RLock()
	factory, exists := r.factories[notifierURL.Scheme]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUnknownScheme, notifierURL.Scheme)
	}

	return factory(notifierURL)
}

// Schemes returns a sorted slice of all registered URL schemes.
func (r *SchemeRegistry) Schemes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return slices.Sorted(maps.Keys(r.factories))
}
