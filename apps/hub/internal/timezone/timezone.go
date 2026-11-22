// Package timezone reads the time zone of a person from the IANA time zone database
// that the binary embeds.
package timezone

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// serverZone names the zone of the machine, which no person can choose.
const serverZone = "Local"

// Parse answers the location that the name gives. It answers errs.ErrTimeZoneUnknown
// for a name that the database does not hold, for an empty name, and for "Local".
func Parse(name string) (*time.Location, error) {
	if name == "" || name == serverZone {
		return nil, fmt.Errorf("loading the time zone %q: %w", name, errs.ErrTimeZoneUnknown)
	}

	location, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf(
			"loading the time zone %q: %w: %w",
			name,
			errs.ErrTimeZoneUnknown,
			err,
		)
	}

	return location, nil
}

// Cache loads the location of each name one time.
type Cache struct {
	logger    *slog.Logger
	locations sync.Map
}

// New creates a Cache that logs each name that no longer loads.
func New(logger *slog.Logger) *Cache {
	return &Cache{logger: logger.With(slog.String("component", "timezone"))}
}

// Of answers the location of the name, or UTC when the name is nil. A record holds
// a name that Parse accepted, so a name that fails here left the database in a later
// release. It computes in UTC then, as a record with no zone does.
func (c *Cache) Of(ctx context.Context, name *string) *time.Location {
	if name == nil {
		return time.UTC
	}

	if cached, ok := c.locations.Load(*name); ok {
		if location, isLocation := cached.(*time.Location); isLocation {
			return location
		}
	}

	location, err := Parse(*name)
	if err != nil {
		c.logger.WarnContext(
			ctx,
			"the time zone of a record no longer loads",
			slog.String("time_zone", *name),
			slog.Any("error", err),
		)

		return time.UTC
	}

	c.locations.Store(*name, location)

	return location
}
