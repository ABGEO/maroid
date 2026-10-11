package notification

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// Kind is one external messaging service. Both maps of a call hold plaintext values,
// keyed by the field key of the model.
type Kind interface {
	ID() string
	Name() string
	// ProviderModel returns a zero value of the struct that declares the fields of a
	// notification provider.
	ProviderModel() any
	// DestinationModel returns a zero value of the struct that declares the fields of
	// a destination.
	DestinationModel() any
	// Verify proves the credential of a notification provider at the service.
	Verify(ctx context.Context, provider map[string]any) error
	// Send delivers one message to one destination.
	Send(ctx context.Context, provider, destination map[string]any, msg pluginapi.Message) error
}

// RefusedError reports that the external messaging service refused a call. Reason
// holds the answer of the service.
type RefusedError struct {
	Reason string
}

// Error returns the reason of the refusal.
func (e *RefusedError) Error() string {
	return "the messaging service refused the call: " + e.Reason
}

// Is reports that a refusal matches errs.ErrDeliveryRefused.
func (e *RefusedError) Is(target error) bool {
	return target == errs.ErrDeliveryRefused
}

// KindEntry is one registered kind with the schema of each of its models.
type KindEntry struct {
	Kind        Kind
	Provider    *settings.Schema
	Destination *settings.Schema
}

// KindRegistry holds the notification provider kinds of the hub.
type KindRegistry struct {
	entries map[string]KindEntry
}

// NewKindRegistry creates a new KindRegistry.
func NewKindRegistry() *KindRegistry {
	return &KindRegistry{entries: make(map[string]KindEntry)}
}

// Register infers the schemas of the kind and stores it. A destination holds no
// secret, so a reader sees it and a route compares it. No field belongs to a user.
func (r *KindRegistry) Register(kind Kind) error {
	if kind == nil {
		return errs.ErrNilNotificationKind
	}

	if _, held := r.entries[kind.ID()]; held {
		return fmt.Errorf("%w: %s", errs.ErrNotificationKindAlreadyRegistered, kind.ID())
	}

	provider, err := inferWorkspaceSchema(kind.ID(), "provider", kind.ProviderModel())
	if err != nil {
		return err
	}

	destination, err := inferWorkspaceSchema(kind.ID(), "destination", kind.DestinationModel())
	if err != nil {
		return err
	}

	if slices.Contains(slices.Collect(maps.Values(destination.Kinds)), model.FieldKindSecret) {
		return fmt.Errorf(
			"%w: the destination of %s declares a secret",
			errs.ErrInvalidNotificationKind,
			kind.ID(),
		)
	}

	r.entries[kind.ID()] = KindEntry{Kind: kind, Provider: provider, Destination: destination}

	return nil
}

// Get returns the kind with the identifier.
func (r *KindRegistry) Get(id string) (KindEntry, bool) {
	entry, found := r.entries[id]

	return entry, found
}

// All returns every kind, in the order of the identifiers.
func (r *KindRegistry) All() []KindEntry {
	entries := make([]KindEntry, 0, len(r.entries))
	for _, id := range slices.Sorted(maps.Keys(r.entries)) {
		entries = append(entries, r.entries[id])
	}

	return entries
}

func inferWorkspaceSchema(kindID string, part string, fields any) (*settings.Schema, error) {
	schema, err := settings.Infer(fields)
	if err != nil {
		return nil, fmt.Errorf("inferring the %s schema of %s: %w", part, kindID, err)
	}

	if slices.Contains(slices.Collect(maps.Values(schema.Scopes)), model.SettingScopeUser) {
		return nil, fmt.Errorf(
			"%w: the %s of %s declares a field of the user",
			errs.ErrInvalidNotificationKind,
			part,
			kindID,
		)
	}

	return schema, nil
}
