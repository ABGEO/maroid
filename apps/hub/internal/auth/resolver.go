package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/timezone"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// The connectors of Dex whose identifier the hub fixes.
const (
	// ProviderTelegram is the connector of Dex that federates Telegram.
	ProviderTelegram = "telegram"
	// ProviderLocal is the connector of Dex that holds the local accounts.
	ProviderLocal = "local"
)

// TimeZoneReader answers the location of the time zone of a user record.
type TimeZoneReader interface {
	TimeZone(ctx context.Context, user *model.User) *time.Location
}

// IdentityResolver reads the user record that one external account names, and the
// time zone of that record.
type IdentityResolver interface {
	TimeZoneReader
	ResolveByProvider(
		ctx context.Context,
		provider string,
		providerUserID string,
	) (*model.User, error)
}

// Resolver is the identity backed implementation of IdentityResolver.
type Resolver struct {
	db    *sqlx.DB
	zones *timezone.Cache
}

var _ IdentityResolver = (*Resolver)(nil)

// NewResolver creates a new Resolver that loads each time zone through zones.
func NewResolver(db *sqlx.DB, zones *timezone.Cache) *Resolver {
	return &Resolver{db: db, zones: zones}
}

// TimeZone answers the location of the time zone of the record, or UTC for none.
func (r *Resolver) TimeZone(ctx context.Context, user *model.User) *time.Location {
	return r.zones.Of(ctx, user.TimeZone)
}

// WithActingUser returns a context that carries the record as the acting user, with
// the location of its time zone.
func WithActingUser(ctx context.Context, zones TimeZoneReader, user *model.User) context.Context {
	return pluginapi.ContextWithActingTimeZone(
		pluginapi.ContextWithActingUser(ctx, user.ID),
		zones.TimeZone(ctx, user),
	)
}

// ResolveByProvider returns the active user record that the external account names.
func (r *Resolver) ResolveByProvider(
	ctx context.Context,
	provider string,
	providerUserID string,
) (*model.User, error) {
	user, err := database.FetchTx(ctx, r.db, func(tx *sqlx.Tx) (*model.User, error) {
		return repository.NewIdentity(tx).GetActiveUserByProvider(ctx, provider, providerUserID)
	})
	if err != nil {
		return nil, fmt.Errorf("resolving the acting user: %w", err)
	}

	return user, nil
}
