package pluginapi_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// PROFILE-SC-017: A context with no zone reads UTC, and a context with Asia/Tbilisi
// reads Asia/Tbilisi.
func TestActingTimeZoneFallsBackToUTC(t *testing.T) {
	t.Parallel()

	tbilisi, err := time.LoadLocation("Asia/Tbilisi")
	require.NoError(t, err)

	require.Equal(t, time.UTC, pluginapi.ActingTimeZone(t.Context()))
	require.Equal(t, time.UTC, pluginapi.ActingTimeZone(
		pluginapi.ContextWithActingTimeZone(t.Context(), nil),
	))
	require.Equal(t, tbilisi, pluginapi.ActingTimeZone(
		pluginapi.ContextWithActingTimeZone(t.Context(), tbilisi),
	))
}
