package rest_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest"
)

const (
	sampleEnvironment = "01a0cae5-eb36-777a-824e-6e7e28d7a6b1"
	sampleRow         = "01a0cae5-eb36-777a-824e-71285629ec17"
)

func sampleCursor() rest.Cursor {
	return rest.Cursor{
		Sort:      []string{"id"},
		Direction: rest.DirectionForward,
		Filters:   map[string]string{environmentFilter: sampleEnvironment},
		Boundary:  map[string]string{"id": sampleRow},
		ID:        sampleRow,
	}
}

// APIFMT-SC-004: A cursor that the hub produced decodes to the position that it
// named, so a client reads the next page from where the last one stopped.
func TestACursorSurvivesTheRoundTrip(t *testing.T) {
	t.Parallel()

	encoded, err := rest.EncodeCursor(sampleCursor())
	require.NoError(t, err)

	decoded, err := rest.DecodeCursor(encoded)
	require.NoError(t, err)
	assert.Equal(t, sampleCursor(), decoded)
}

// RES-006: A cursor is base64url, so it travels in a query string with no
// escaping, and a client reads nothing from it.
func TestACursorIsBase64URL(t *testing.T) {
	t.Parallel()

	encoded, err := rest.EncodeCursor(sampleCursor())
	require.NoError(t, err)

	assert.NotContains(t, encoded, "+")
	assert.NotContains(t, encoded, "/")
	assert.NotContains(t, encoded, "=")

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "direction")
}

// APIFMT-SC-007: A cursor that Maroid did not produce answers a failure, and
// never a page from a position that nothing named.
func TestACursorThatMaroidDidNotProduceFails(t *testing.T) {
	t.Parallel()

	for name, value := range map[string]string{
		"not base64":    "!!!not-base64!!!",
		"not json":      base64.RawURLEncoding.EncodeToString([]byte("plain text")),
		"another shape": base64.RawURLEncoding.EncodeToString([]byte(`{"unexpected":1}`)),
		"empty":         "",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := rest.DecodeCursor(value)
			require.Error(t, err)
		})
	}
}

// APIFMT-SC-007: A cursor built under one set of filters, sent with another, is
// stale. A client tells that from a cursor it built wrong.
func TestAStaleCursorIsTheOneThatDoesNotMatchTheRequest(t *testing.T) {
	t.Parallel()

	held := sampleCursor()

	assert.True(t, held.Matches([]string{"id"},
		map[string]string{environmentFilter: sampleEnvironment}))

	assert.False(t, held.Matches([]string{"id"}, map[string]string{environmentFilter: "another"}),
		"another filter makes it stale")
	assert.False(t, held.Matches([]string{"name"},
		map[string]string{environmentFilter: sampleEnvironment}),
		"another sort makes it stale")
	assert.False(t, held.Matches([]string{"id"}, nil), "a dropped filter makes it stale")
}

// RES-006: A cursor carries no signature, so nothing in it needs a secret. It
// still refuses a value whose shape is not a cursor.
func TestACursorCarriesNoSecret(t *testing.T) {
	t.Parallel()

	encoded, err := rest.EncodeCursor(sampleCursor())
	require.NoError(t, err)

	assert.NotContains(t, encoded, ".", "no signature separator")
}
