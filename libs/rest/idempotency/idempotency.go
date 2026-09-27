// Package idempotency answers a repeated write with the result of the first one.
package idempotency

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/abgeo/maroid/libs/rest/problem"
)

const (
	// KeyHeader carries the key that a client picks for one write, so
	// that repeating that write is safe.
	KeyHeader = "Idempotency-Key"

	idempotencyKeyMaxLength = 255
	idempotencyBodyMax      = 1 << 20
	storeTimeout            = 5 * time.Second
)

// ErrInProgress reports a key whose first write still runs. The repeat answers
// 409 and the client sends it again later.
var ErrInProgress = errors.New("idempotency: a write under the key still runs")

// Answer is the answer that one write produced, kept so that a repeat
// of that write reads it again.
type Answer struct {
	RequestHash string
	Status      int
	Header      http.Header
	Body        []byte
}

// Store keeps the answer of a write under the key that a client picked. The
// hub implements it over a table, and this module holds no database.
//
// A write claims its key before it runs, so two requests under one key never
// both run the write, even when the second arrives while the first still runs.
type Store interface {
	// Reserve claims the key for a request whose hash is hash. It answers nil
	// when this request claimed the key and runs the write, the answer that a
	// finished write left, or ErrInProgress while the first write still runs.
	Reserve(ctx context.Context, key string, hash string) (*Answer, error)
	// Complete stores the answer of the write that claimed the key.
	Complete(ctx context.Context, key string, answer Answer) error
	// Release frees a key whose write failed, so that a repeat runs it again.
	Release(ctx context.Context, key string) error
}

// Middleware answers a repeated write with the result of the first one.
//
// It runs behind the access check, because the store scopes every row to the
// acting user and a request with no acting user reaches no row. A request that
// carries no key passes through, as does a method that creates nothing. A
// keyed body larger than 1 MiB answers 413, and a repeat that arrives while
// the first write still runs answers 409.
func Middleware(logger *slog.Logger, store Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimSpace(r.Header.Get(KeyHeader))
			if key == "" || r.Method != http.MethodPost {
				next.ServeHTTP(w, r)

				return
			}

			if len(key) > idempotencyKeyMaxLength {
				problem.Write(w, r, problem.NewRequestInvalid().
					WithDetail("The Idempotency-Key header is longer than the store holds."))

				return
			}

			body, err := io.ReadAll(io.LimitReader(r.Body, idempotencyBodyMax+1))
			if err != nil {
				problem.Write(w, r, problem.NewRequestInvalid().
					WithDetail("The body of the request did not arrive in full."))

				return
			}

			// The cache compares the whole body before the write runs, so it
			// holds the body in memory. A larger body under a key would lose the
			// promise of the key without a word, so it answers a failure.
			if len(body) > idempotencyBodyMax {
				problem.Write(w, r, problem.NewContentTooLarge().
					WithDetail("A body under an Idempotency-Key holds at most 1 MiB."))

				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			hash := digestOf(r, body)

			held, err := store.Reserve(r.Context(), key, hash)

			switch {
			case errors.Is(err, ErrInProgress):
				w.Header().Set("Retry-After", "1")
				problem.Write(w, r, problem.NewRequestInProgress())

				return
			case err != nil:
				// A cache that fails leaves the write unguarded rather than
				// refusing it. That is the safe direction for the person, and it
				// is invisible without this line.
				logger.ErrorContext(r.Context(), "the key cache claimed nothing",
					slog.String("idempotency_key", key), slog.Any("error", err))
				next.ServeHTTP(w, r)

				return
			case held != nil:
				replay(w, r, *held, hash)

				return
			}

			run(w, r, next, logger, store, key, hash)
		})
	}
}

// run serves the write that claimed the key, then stores its answer, or frees
// the key when the write failed.
//
// The client that lost its answer is the one whose request context ends, and it
// is the client that repeats. The write has landed, so the store runs past that
// end. A handler that panics frees the key before the panic travels on.
func run(
	w http.ResponseWriter,
	r *http.Request,
	next http.Handler,
	logger *slog.Logger,
	store Store,
	key string,
	hash string,
) {
	storeCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), storeTimeout)
	defer cancel()

	settled := false

	defer func(ctx context.Context) {
		if settled {
			return
		}

		if err := store.Release(ctx, key); err != nil {
			logger.ErrorContext(ctx, "the key cache freed no key",
				slog.String("idempotency_key", key), slog.Any("error", err))
		}
	}(storeCtx)

	recorder := &recordedAnswer{ResponseWriter: w, status: http.StatusOK}
	next.ServeHTTP(recorder, r)

	// A handler that writes nothing answers 200 when it returns, with the
	// headers it set. The recorder saw no write, so it reads them now.
	if !recorder.written {
		recorder.header = w.Header().Clone()
	}

	if recorder.status >= http.StatusBadRequest {
		return
	}

	settled = true

	if err := store.Complete(storeCtx, key, Answer{
		RequestHash: hash,
		Status:      recorder.status,
		Header:      replayable(recorder.header),
		Body:        recorder.body.Bytes(),
	}); err != nil {
		logger.ErrorContext(r.Context(), "the key cache kept no answer",
			slog.String("idempotency_key", key), slog.Any("error", err))
	}
}

// replay answers the stored result. A request whose hash differs is another
// request under the same key, and it answers 400.
func replay(w http.ResponseWriter, r *http.Request, held Answer, hash string) {
	if held.RequestHash != hash {
		problem.Write(w, r, problem.NewRequestInvalid().WithDetail(
			"The Idempotency-Key header names an earlier request that differs from this one.",
		))

		return
	}

	// A header that this request already carries stays: the chain sets the flow
	// identifier and the cache period for this request, not for the first one.
	for name, values := range replayable(held.Header) {
		if w.Header().Get(name) != "" {
			continue
		}

		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	w.WriteHeader(held.Status)
	_, _ = w.Write(held.Body)
}

// replayable keeps the headers that describe the answer itself. A cookie, a
// CORS grant, or any other header belongs to the request that first carried it,
// and a replay must never hand it to another.
func replayable(header http.Header) http.Header {
	names := []string{
		"Content-Type",
		"Location",
		"ETag",
	}
	kept := http.Header{}

	for _, name := range names {
		if values := header.Values(name); len(values) > 0 {
			kept[http.CanonicalHeaderKey(name)] = values
		}
	}

	return kept
}

// digestOf builds the request hash that the cache compares. The query counts,
// because a route may carry its input there and not in the body.
func digestOf(r *http.Request, body []byte) string {
	sum := sha256.New()
	sum.Write([]byte(r.Method))
	sum.Write([]byte{0})
	sum.Write([]byte(r.URL.Path))
	sum.Write([]byte{0})
	sum.Write([]byte(r.URL.Query().Encode()))
	sum.Write([]byte{0})
	sum.Write(body)

	return hex.EncodeToString(sum.Sum(nil))
}

// recordedAnswer keeps what a handler wrote, so that the middleware stores it.
type recordedAnswer struct {
	http.ResponseWriter

	status  int
	written bool
	header  http.Header
	body    bytes.Buffer
}

func (a *recordedAnswer) WriteHeader(status int) {
	if a.written {
		return
	}

	a.status = status
	a.written = true
	a.header = a.ResponseWriter.Header().Clone()

	a.ResponseWriter.WriteHeader(status)
}

// Unwrap answers the writer that this one records for, so that
// http.ResponseController reaches it.
func (a *recordedAnswer) Unwrap() http.ResponseWriter {
	return a.ResponseWriter
}

func (a *recordedAnswer) Write(payload []byte) (int, error) {
	if !a.written {
		a.WriteHeader(http.StatusOK)
	}

	a.body.Write(payload)

	//nolint:wrapcheck // the wrapped writer owns the error of the write.
	return a.ResponseWriter.Write(payload)
}
