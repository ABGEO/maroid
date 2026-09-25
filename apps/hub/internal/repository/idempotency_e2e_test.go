package repository_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/abgeo/maroid/apps/hub/internal/idempotency"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest"
)

// APIFMT-SC-020: The middleware and the store of the hub answer a repeat with
// the first result. The two are tested apart elsewhere, and this joins them.
func TestARepeatedWriteReachesTheHandlerOnce(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	user := insertUser(t, instance, nameOfA)

	store := idempotency.NewStore(instance.DB)

	var runs atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		runs.Add(1)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"01a0cae5-eb36-777a-824e-6e7e28d7a6b1"}`))
	})

	send := func() *httptest.ResponseRecorder {
		request := httptest.NewRequestWithContext(
			pluginapi.ContextWithActingUser(t.Context(), user),
			http.MethodPost,
			"/plugins/dev.maroid.jasmine/api/environments",
			strings.NewReader(`{"name":"Balcony"}`),
		)
		request.Header.Set(rest.IdempotencyKeyHeader, "key-1")

		recorder := httptest.NewRecorder()
		rest.Idempotency(slog.New(slog.DiscardHandler), store)(handler).ServeHTTP(recorder, request)

		return recorder
	}

	first := send()
	second := send()

	assert.Equal(t, http.StatusCreated, first.Code)
	assert.Equal(t, http.StatusCreated, second.Code)
	assert.Equal(t, "application/json", second.Header().Get("Content-Type"))
	assert.JSONEq(t, first.Body.String(), second.Body.String())
	assert.Equal(t, int32(1), runs.Load(), "the handler ran once")
}

// APIFMT-SC-020, OWN-006: One key belongs to one person. Two people who pick
// the same key never read each other's answer, and each write runs once for
// each of them.
func TestOneKeyOfTwoPeopleAnswersEachOfThem(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)
	userB := insertUser(t, instance, nameOfB)

	store := idempotency.NewStore(instance.DB)

	var runs atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runs.Add(1)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"owner":"` + pluginapi.ActingUserFromContext(r.Context()) + `"}`))
	})

	send := func(user string) *httptest.ResponseRecorder {
		request := httptest.NewRequestWithContext(
			pluginapi.ContextWithActingUser(t.Context(), user),
			http.MethodPost,
			"/plugins/dev.maroid.jasmine/api/environments",
			strings.NewReader(`{"name":"Balcony"}`),
		)
		// The same key, picked by two different people.
		request.Header.Set(rest.IdempotencyKeyHeader, "shared-key")

		recorder := httptest.NewRecorder()
		rest.Idempotency(slog.New(slog.DiscardHandler), store)(handler).ServeHTTP(recorder, request)

		return recorder
	}

	firstOfA := send(userA)
	firstOfB := send(userB)

	assert.Equal(t, int32(2), runs.Load(), "the key of one person never answers another")
	assert.Contains(t, firstOfA.Body.String(), userA)
	assert.Contains(t, firstOfB.Body.String(), userB,
		"the answer of the first person never reaches the second")

	// Each of them repeats, and each reads their own answer.
	assert.JSONEq(t, firstOfA.Body.String(), send(userA).Body.String())
	assert.JSONEq(t, firstOfB.Body.String(), send(userB).Body.String())
	assert.Equal(t, int32(2), runs.Load(), "neither repeat reached the handler")
}
