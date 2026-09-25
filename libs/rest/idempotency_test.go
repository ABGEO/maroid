package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest"
)

// memoryStore is the key cache of one test, and of one person. A real store
// answers each person their own row, which the store of the hub does in the
// policy of its table.
type memoryStore struct {
	held map[string]rest.IdempotentAnswer
}

func newMemoryStore() *memoryStore {
	return &memoryStore{held: map[string]rest.IdempotentAnswer{}}
}

func (s *memoryStore) Answer(_ context.Context, key string) (rest.IdempotentAnswer, error) {
	answer, found := s.held[key]
	if !found {
		return rest.IdempotentAnswer{}, rest.ErrNoIdempotentAnswer
	}

	return answer, nil
}

func (s *memoryStore) Keep(_ context.Context, key string, answer rest.IdempotentAnswer) error {
	s.held[key] = answer

	return nil
}

// creator counts how often the handler behind the middleware ran.
type creator struct {
	runs atomic.Int32
}

func (c *creator) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	c.runs.Add(1)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(`{"id":"01a0cae5-eb36-777a-824e-6e7e28d7a6b1"}`))
}

// writeWithKey sends one write through the middleware.
func writeWithKey(
	t *testing.T,
	store rest.IdempotencyStore,
	handler http.Handler,
	key string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/plants", strings.NewReader(body),
	)
	if key != "" {
		request.Header.Set(rest.IdempotencyKeyHeader, key)
	}

	recorder := httptest.NewRecorder()
	rest.Idempotency(slog.New(slog.DiscardHandler), store)(handler).ServeHTTP(recorder, request)

	return recorder
}

// APIFMT-SC-020: A client repeats a write it did not see answered, and one
// record exists. Both answers hold one body.
func TestARepeatedWriteAnswersTheEarlierResult(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	handler := &creator{}

	first := writeWithKey(t, store, handler, "key-1", `{"name":"Jasmine"}`)
	second := writeWithKey(t, store, handler, "key-1", `{"name":"Jasmine"}`)

	assert.Equal(t, http.StatusCreated, first.Code)
	assert.Equal(t, http.StatusCreated, second.Code)
	assert.JSONEq(t, first.Body.String(), second.Body.String())
	assert.Equal(t, int32(1), handler.runs.Load(), "the handler ran once")
}

// APIFMT-SC-020: A second send under one key with another body answers 400,
// which Z-230 recommends, because the key names another request.
func TestARepeatUnderOneKeyWithAnotherBodyIsRefused(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	handler := &creator{}

	writeWithKey(t, store, handler, "key-1", `{"name":"Jasmine"}`)
	second := writeWithKey(t, store, handler, "key-1", `{"name":"Fern"}`)

	require.Equal(t, http.StatusBadRequest, second.Code)
	assert.Equal(t, int32(1), handler.runs.Load(), "the handler did not run again")
}

// APIFMT-SC-020: A repeat under a new key makes a second record.
func TestAWriteUnderANewKeyRunsAgain(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	handler := &creator{}

	writeWithKey(t, store, handler, "key-1", `{"name":"Jasmine"}`)
	writeWithKey(t, store, handler, "key-2", `{"name":"Jasmine"}`)

	assert.Equal(t, int32(2), handler.runs.Load())
}

// APIFMT-SC-020: A write that names no key passes through, because Z-230 makes
// the header optional.
func TestAWriteWithNoKeyIsNeverStored(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	handler := &creator{}

	writeWithKey(t, store, handler, "", `{"name":"Jasmine"}`)
	writeWithKey(t, store, handler, "", `{"name":"Jasmine"}`)

	assert.Equal(t, int32(2), handler.runs.Load())
	assert.Empty(t, store.held, "the cache holds nothing")
}

// APIFMT-SC-020: A write that failed is never stored, so a client may retry it
// and reach the handler again.
func TestAFailedWriteIsNeverStored(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()

	var runs atomic.Int32

	failing := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runs.Add(1)
		rest.Write(w, r, rest.NewInternal())
	})

	writeWithKey(t, store, failing, "key-1", `{"name":"Jasmine"}`)
	writeWithKey(t, store, failing, "key-1", `{"name":"Jasmine"}`)

	assert.Equal(t, int32(2), runs.Load(), "a failure leaves the key free")
	assert.Empty(t, store.held)
}

// APIFMT-SC-020: A body larger than the cache holds reaches the handler whole.
// The middleware buffers to hash, and a truncated buffer would hand the handler
// a body that the client never sent.
func TestABodyTooLargeToCacheReachesTheHandlerWhole(t *testing.T) {
	t.Parallel()

	const oversized = (1 << 20) + 4096

	store := newMemoryStore()
	sent := strings.Repeat("x", oversized)

	var read atomic.Int64

	// A require inside a handler runs off the test goroutine, so the handler
	// records what it saw and the test reads it afterwards.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			read.Store(-1)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		read.Store(int64(len(body)))
		w.WriteHeader(http.StatusCreated)
	})

	recorder := writeWithKey(t, store, handler, "key-1", sent)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, int64(oversized), read.Load(), "the handler read every byte")
	assert.Empty(t, store.held, "the cache holds no body this large")
}

// APIFMT-SC-020: A body that never arrives answers a failure that names the
// request, because nothing is known about the shape of a body that was not read.
func TestABodyThatDoesNotArriveAnswersARequestFailure(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/plants", errReader{},
	)
	request.Header.Set(rest.IdempotencyKeyHeader, "key-1")

	recorder := httptest.NewRecorder()
	rest.Idempotency(slog.New(slog.DiscardHandler), newMemoryStore())(&creator{}).
		ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)

	var problem rest.Problem

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &problem))
	assert.Equal(t, rest.TypeRequestInvalid, problem.Type)
	assert.NotEqual(t, rest.TypeBodyInvalid, problem.Type,
		"the shape of a body that never arrived is unknown")
}

// errReader is a body that fails part way through.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errNoBody }

var errNoBody = errors.New("the connection went")

// APIFMT-SC-020: A repeat answers what the first write answered, headers and
// all. A body with no content type is sniffed, and JSON sniffs as text.
func TestARepeatAnswersTheContentTypeOfTheFirstWrite(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	handler := &creator{}

	first := writeWithKey(t, store, handler, "key-1", `{"name":"Jasmine"}`)
	second := writeWithKey(t, store, handler, "key-1", `{"name":"Jasmine"}`)

	assert.Equal(t, "application/json", first.Header().Get("Content-Type"))
	assert.Equal(t, first.Header().Get("Content-Type"), second.Header().Get("Content-Type"),
		"a repeat answers the media type that the first write answered")
}
