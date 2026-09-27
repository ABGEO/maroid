package rest_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest"
)

// APIFMT-SC-019: The value that the client reads off ETag is the value that the
// hub reads back out of If-Match. The two halves are written apart, so this
// pins the one string that joins them.
func TestTheEntityTagOfAReadIsReadableAsAnIfMatch(t *testing.T) {
	t.Parallel()

	moment := time.Date(2026, time.September, 25, 10, 30, 0, 123456789, time.UTC)

	// What a read answers, and what the deck now holds in its form.
	answered := rest.ETag(moment)
	require.Equal(t, `W/"1790332200123456789"`, answered)

	// The same string, arriving on the write that the client sends back.
	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodPut, "/plugins/p/settings", nil,
	)
	request.Header.Set(rest.IfMatchHeader, answered)

	read, held, err := rest.IfMatch(request)
	require.NoError(t, err)
	assert.True(t, held)
	assert.True(t, moment.Equal(read), "the hub reads back the moment that the read answered")
}
