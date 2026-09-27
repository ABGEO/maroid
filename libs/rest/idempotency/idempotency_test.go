package idempotency_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/idempotency"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// memoryStore is the key cache of one test, and of one person. A real store
// answers each person their own row, which the store of the hub does in the
// policy of its table. held holds the finished answers, and pending the keys
// whose first write still runs.
type memoryStore struct {
	mu      sync.Mutex
	held    map[string]idempotency.Answer
	pending map[string]bool
}

func newMemoryStore() *memoryStore {
	return &memoryStore{held: map[string]idempotency.Answer{}, pending: map[string]bool{}}
}

func (s *memoryStore) Reserve(
	_ context.Context,
	key string,
	_ string,
) (*idempotency.Answer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if answer, found := s.held[key]; found {
		return &answer, nil
	}

	if s.pending[key] {
		return nil, idempotency.ErrInProgress
	}

	s.pending[key] = true

	return nil, nil //nolint:nilnil // a claimed key holds no answer yet.
}

func (s *memoryStore) Complete(_ context.Context, key string, answer idempotency.Answer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.pending, key)
	s.held[key] = answer

	return nil
}

func (s *memoryStore) Release(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.pending, key)

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
	store idempotency.Store,
	handler http.Handler,
	key string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/plants", strings.NewReader(body),
	)
	if key != "" {
		request.Header.Set(idempotency.KeyHeader, key)
	}

	recorder := httptest.NewRecorder()
	idempotency.Middleware(slog.New(slog.DiscardHandler), store)(
		handler,
	).ServeHTTP(recorder, request)

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
		problem.Write(w, r, problem.NewInternal())
	})

	writeWithKey(t, store, failing, "key-1", `{"name":"Jasmine"}`)
	writeWithKey(t, store, failing, "key-1", `{"name":"Jasmine"}`)

	assert.Equal(t, int32(2), runs.Load(), "a failure leaves the key free")
	assert.Empty(t, store.held)
	assert.Empty(t, store.pending, "a failure leaves no claim")
}

// APIFMT-SC-020: A keyed body larger than the cache holds answers 413 and never
// reaches the handler, because the key could not keep its promise. The same
// body with no key reaches the handler whole.
func TestABodyTooLargeToCacheIsRefusedUnderAKey(t *testing.T) {
	t.Parallel()

	const oversized = (1 << 20) + 4096

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

	store := newMemoryStore()
	keyed := writeWithKey(t, store, handler, "key-1", sent)

	assert.Equal(t, http.StatusRequestEntityTooLarge, keyed.Code)
	assert.Contains(t, keyed.Body.String(), problem.TypeContentTooLarge)
	assert.Zero(t, read.Load(), "the handler never ran")
	assert.Empty(t, store.held)

	unkeyed := writeWithKey(t, store, handler, "", sent)

	assert.Equal(t, http.StatusCreated, unkeyed.Code)
	assert.Equal(t, int64(oversized), read.Load(), "the handler read every byte")
}

// APIFMT-SC-020: A body that never arrives answers a failure that names the
// request, because nothing is known about the shape of a body that was not read.
func TestABodyThatDoesNotArriveAnswersARequestFailure(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/plants", errReader{},
	)
	request.Header.Set(idempotency.KeyHeader, "key-1")

	recorder := httptest.NewRecorder()
	idempotency.Middleware(slog.New(slog.DiscardHandler), newMemoryStore())(&creator{}).
		ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)

	var answered problem.Problem

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answered))
	assert.Equal(t, problem.TypeRequestInvalid, answered.Type)
	assert.NotEqual(t, problem.TypeBodyInvalid, answered.Type,
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

// contextStore refuses to keep an answer under a context that has ended, as a
// database store does.
type contextStore struct {
	*memoryStore
}

func (s contextStore) Complete(ctx context.Context, key string, answer idempotency.Answer) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("keeping the answer: %w", err)
	}

	return s.memoryStore.Complete(ctx, key, answer)
}

// APIFMT-SC-020: The client that loses its answer drops the connection, which
// ends the request context after the write lands. The answer is still kept, so
// the repeat of that client answers it and one record exists.
func TestAWriteWhoseClientLeftStillKeepsTheAnswer(t *testing.T) {
	t.Parallel()

	store := contextStore{newMemoryStore()}
	handler := &creator{}

	ctx, disconnect := context.WithCancel(t.Context())
	request := httptest.NewRequestWithContext(
		ctx, http.MethodPost, "/plants", strings.NewReader(`{"name":"Fern"}`),
	)
	request.Header.Set(idempotency.KeyHeader, "left-early")

	idempotency.Middleware(slog.New(slog.DiscardHandler), store)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r)
			disconnect()
		}),
	).ServeHTTP(httptest.NewRecorder(), request)

	repeat := writeWithKey(t, store, handler, "left-early", `{"name":"Fern"}`)

	assert.Equal(t, http.StatusCreated, repeat.Code)
	assert.Equal(t, int32(1), handler.runs.Load(), "the repeat answers the kept result")
}

// sendTo runs one keyed write at target through the middleware.
func sendTo(
	t *testing.T,
	store idempotency.Store,
	handler http.Handler,
	target string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, target, strings.NewReader(`{"name":"Fern"}`),
	)
	request.Header.Set(idempotency.KeyHeader, "key-1")

	recorder := httptest.NewRecorder()
	idempotency.Middleware(slog.New(slog.DiscardHandler), store)(
		handler,
	).ServeHTTP(recorder, request)

	return recorder
}

// APIFMT-SC-020: A repeat answers the headers that describe the answer, and no
// other. A cookie or a CORS grant belongs to the request that first carried it,
// so the cache neither keeps it nor hands it to a repeat.
func TestARepeatReplaysOnlyTheHeadersOfTheAnswer(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Location", "/plants/1")
		w.Header().Set("ETag", `"1790332200123456789"`)
		w.Header().Set("Set-Cookie", "__Host-maroid_binding=secret")
		w.Header().Set("Access-Control-Allow-Origin", "https://first.example")
		w.Header().Set("X-Probe", "first")
		w.WriteHeader(http.StatusCreated)
	})

	sendTo(t, store, handler, "/plants")
	repeat := sendTo(t, store, handler, "/plants")

	assert.Equal(t, http.StatusCreated, repeat.Code)
	assert.Equal(t, "application/json", repeat.Header().Get("Content-Type"))
	assert.Equal(t, "/plants/1", repeat.Header().Get("Location"))
	assert.Equal(t, `"1790332200123456789"`, repeat.Header().Get("ETag"))

	for _, name := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "X-Probe"} {
		assert.Empty(t, repeat.Header().Get(name), "a repeat never carries %s", name)
		assert.Empty(t, store.held["key-1"].Header.Get(name), "the cache never keeps %s", name)
	}
}

// APIFMT-SC-020: A row that an earlier release stored with every header still
// replays the headers of the answer alone.
func TestAStoredCookieNeverReplays(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	handler := &creator{}

	sendTo(t, store, handler, "/plants")

	held := store.held["key-1"]
	held.Header.Set("Set-Cookie", "__Host-maroid_binding=secret")
	store.held["key-1"] = held

	repeat := sendTo(t, store, handler, "/plants")

	assert.Equal(t, http.StatusCreated, repeat.Code)
	assert.Empty(t, repeat.Header().Get("Set-Cookie"))
}

// APIFMT-SC-020: A route may carry its input in the query. The same key with
// another query is another request, so it answers 400 and never the first
// answer.
func TestARepeatUnderOneKeyWithAnotherQueryIsRefused(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	handler := &creator{}

	first := sendTo(t, store, handler, "/identities?provider=telegram")
	other := sendTo(t, store, handler, "/identities?provider=cloud")

	assert.Equal(t, http.StatusCreated, first.Code)
	assert.Equal(t, http.StatusBadRequest, other.Code)
	assert.Equal(t, int32(1), handler.runs.Load())

	same := sendTo(t, store, handler, "/identities?provider=telegram")
	assert.Equal(t, http.StatusCreated, same.Code)
	assert.Equal(t, int32(1), handler.runs.Load(), "the same query replays")
}

// APIFMT-SC-020, APIFMT-FR-020: A client that times out sends its write again
// while the first still runs. The repeat answers 409 and never runs the write,
// and once the first write lands the next repeat reads its answer. One record
// exists.
func TestARepeatWhileTheFirstWriteRunsAnswersConflict(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	started := make(chan struct{})
	finish := make(chan struct{})

	var runs atomic.Int32

	slow := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		runs.Add(1)
		close(started)
		<-finish

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"first"}`))
	})

	firstDone := make(chan *httptest.ResponseRecorder)

	go func() { firstDone <- writeWithKey(t, store, slow, "key-1", `{"name":"Fern"}`) }()

	<-started

	during := writeWithKey(t, store, slow, "key-1", `{"name":"Fern"}`)

	assert.Equal(t, http.StatusConflict, during.Code)
	assert.Equal(t, "1", during.Header().Get("Retry-After"))
	assert.Contains(t, during.Body.String(), problem.TypeRequestInProgress)

	close(finish)

	first := <-firstDone
	after := writeWithKey(t, store, slow, "key-1", `{"name":"Fern"}`)

	assert.Equal(t, http.StatusCreated, first.Code)
	assert.Equal(t, http.StatusCreated, after.Code)
	assert.JSONEq(t, first.Body.String(), after.Body.String())
	assert.Equal(t, int32(1), runs.Load(), "the write ran once")
}

// APIFMT-SC-020: A handler that panics leaves no claim on its key, so the
// repeat runs the write rather than answering 409 until the claim lapses.
func TestAPanickingWriteFreesItsKey(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()

	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("the handler gave up")
	})

	assert.Panics(t, func() {
		writeWithKey(t, store, panicking, "key-1", `{"name":"Fern"}`)
	})

	repeat := writeWithKey(t, store, &creator{}, "key-1", `{"name":"Fern"}`)
	assert.Equal(t, http.StatusCreated, repeat.Code)
}

// A keyed handler reaches the writer of the server through
// http.ResponseController, so it can flush or set a deadline.
func TestAKeyedHandlerReachesTheResponseController(t *testing.T) {
	t.Parallel()

	var flushErr atomic.Value

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		flushErr.Store(fmt.Sprint(http.NewResponseController(w).Flush()))
	})

	recorder := writeWithKey(t, newMemoryStore(), handler, "key-1", `{"name":"Fern"}`)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "<nil>", flushErr.Load(), "the flush reaches the server")
}

// APIFMT-SC-020: A handler that sets its headers and writes nothing answers
// 200 when it returns. A repeat answers the same headers.
func TestARepeatOfAnAnswerWithNoBodyKeepsItsHeaders(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()

	var runs atomic.Int32

	silent := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		runs.Add(1)
		w.Header().Set("Location", "/plants/1")
	})

	first := writeWithKey(t, store, silent, "key-1", `{"name":"Fern"}`)
	repeat := writeWithKey(t, store, silent, "key-1", `{"name":"Fern"}`)

	assert.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, http.StatusOK, repeat.Code)
	assert.Equal(t, "/plants/1", repeat.Header().Get("Location"))
	assert.Equal(t, int32(1), runs.Load())
}
