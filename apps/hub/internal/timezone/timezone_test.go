package timezone_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/timezone"
)

// PROFILE-SC-013: Parse accepts a name of the IANA time zone database, "UTC" included.
func TestParseAcceptsANameOfTheDatabase(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"Asia/Tbilisi", "UTC", "Europe/Berlin"} {
		location, err := timezone.Parse(name)
		require.NoError(t, err, name)
		require.Equal(t, name, location.String())
	}
}

// PROFILE-SC-014: Parse refuses an offset, a misspelled name, and the zone of the server.
func TestParseRefusesANameOutsideTheDatabase(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"+04:00", "Asia/Tbilsi", "Local", ""} {
		_, err := timezone.Parse(name)
		require.ErrorIs(t, err, errs.ErrTimeZoneUnknown, name)
	}
}

// PROFILE-SC-018: A record with no zone, and a name that no longer loads, compute in UTC.
func TestCacheAnswersUTCWithNoZone(t *testing.T) {
	t.Parallel()

	cache := timezone.New(slog.New(slog.DiscardHandler))
	tbilisi := "Asia/Tbilisi"
	gone := "Mars/Olympus_Mons"

	require.Equal(t, time.UTC, cache.Of(t.Context(), nil))
	first := cache.Of(t.Context(), &tbilisi)
	second := cache.Of(t.Context(), &tbilisi)

	require.Equal(t, tbilisi, first.String())
	require.Same(t, first, second, "the cache loads a name one time")
	require.Equal(t, time.UTC, cache.Of(t.Context(), &gone))
}
