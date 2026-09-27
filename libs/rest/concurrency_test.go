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

// writtenAt is the moment of the last write of a record under test.
func writtenAt() time.Time {
	return time.Date(2026, time.September, 25, 10, 30, 0, 123456789, time.UTC)
}

// writeWith runs one write through the middleware and answers the moment that
// reached the handler, together with the status that the client saw. A nil moment
// with a 200 is the write that named no validator.
func writeWith(t *testing.T, validator string) (*time.Time, int) {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/plugins", nil)
	if validator != "" {
		request.Header.Set(rest.IfMatchHeader, validator)
	}

	var reached *time.Time

	recorder := httptest.NewRecorder()
	rest.IfMatch(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		reached = rest.IfMatchFromContext(r.Context())
	})).ServeHTTP(recorder, request)

	return reached, recorder.Code
}

// APIFMT-SC-019: The validator comes from the moment of the last write, so no
// column and no migration carries a version. APIFMT-DD-014.
func TestTheEntityTagIsAWeakValidatorOfTheLastWrite(t *testing.T) {
	t.Parallel()

	tag := rest.ETag(writtenAt())

	assert.Equal(t, `W/"1790332200123456789"`, tag)
	assert.True(t, len(tag) > 2 && tag[:2] == "W/", "the validator is weak")
}

// APIFMT-SC-019: Two moments that differ answer two validators, and one moment
// answers one, whatever zone it carries.
func TestTheEntityTagFollowsTheInstant(t *testing.T) {
	t.Parallel()

	zone := time.FixedZone("somewhere", 4*60*60)

	assert.Equal(t, rest.ETag(writtenAt()), rest.ETag(writtenAt().In(zone)),
		"one instant answers one validator in any zone")
	assert.NotEqual(t, rest.ETag(writtenAt()), rest.ETag(writtenAt().Add(time.Nanosecond)))
}

// APIFMT-SC-019: A write that carries the validator it read reaches its handler
// with the moment to compare, and one that carries none reaches it with no
// moment and lands. Z-182 makes the header optional, and APIFMT-FR-019 allows
// the write.
func TestIfMatchCarriesTheValidatorToTheHandler(t *testing.T) {
	t.Parallel()

	moment, status := writeWith(t, rest.ETag(writtenAt()))
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, moment)
	assert.True(t, writtenAt().Equal(*moment))

	moment, status = writeWith(t, "")
	require.Equal(t, http.StatusOK, status)
	assert.Nil(t, moment, "a write with no If-Match lands unconditionally")
}

// APIFMT-SC-019: A strong validator of the same shape reads too, because a
// client that drops the weakness prefix still names one moment.
func TestIfMatchReadsAStrongValidator(t *testing.T) {
	t.Parallel()

	moment, status := writeWith(t, `"1790332200123456789"`)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, moment)
	assert.True(t, writtenAt().Equal(*moment))
}

// APIFMT-SC-019: A value that Maroid did not answer never reaches a handler, so
// no write compares against a moment that nothing produced. The middleware
// answers it, so no route repeats the check.
func TestIfMatchRefusesAValueThatMaroidDidNotAnswer(t *testing.T) {
	t.Parallel()

	for name, value := range map[string]string{
		"no quotes":      "1790332200123456789",
		"not a number":   `W/"yesterday"`,
		"empty quotes":   `W/""`,
		"any":            "*",
		"two validators": `W/"1", W/"2"`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			moment, status := writeWith(t, value)
			assert.Equal(t, http.StatusBadRequest, status)
			assert.Nil(t, moment, "the handler never ran")
		})
	}
}
