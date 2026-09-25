package rest

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
)

const (
	// IdempotencyKeyHeader carries the key that a client picks for one write, so
	// that repeating that write is safe.
	IdempotencyKeyHeader = "Idempotency-Key"

	idempotencyKeyMaxLength = 255
	idempotencyBodyMax      = 1 << 20
)

// ErrNoIdempotentAnswer reports that the store holds no answer for a key.
var ErrNoIdempotentAnswer = errors.New("the store holds no answer for this key")

// IdempotentAnswer is the answer that one write produced, kept so that a repeat
// of that write reads it again.
type IdempotentAnswer struct {
	RequestHash string
	Status      int
	Header      http.Header
	Body        []byte
}

// IdempotencyStore keeps the answer of a write under the key that a client
// picked. The hub implements it over a table, and this module holds no database.
type IdempotencyStore interface {
	// Answer returns what the key holds, or ErrNoIdempotentAnswer.
	Answer(ctx context.Context, key string) (IdempotentAnswer, error)
	// Keep stores the answer of a write under the key.
	Keep(ctx context.Context, key string, answer IdempotentAnswer) error
}

// Idempotency answers a repeated write with the result of the first one.
//
// It runs behind the access check, because the store scopes every row to the
// acting user and a request with no acting user reaches no row. A request that
// carries no key passes through, as does a method that creates nothing.
func Idempotency(logger *slog.Logger, store IdempotencyStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimSpace(r.Header.Get(IdempotencyKeyHeader))
			if key == "" || r.Method != http.MethodPost {
				next.ServeHTTP(w, r)

				return
			}

			if len(key) > idempotencyKeyMaxLength {
				Write(w, r, NewRequestInvalid().
					WithDetail("The Idempotency-Key header is longer than the store holds."))

				return
			}

			body, err := io.ReadAll(io.LimitReader(r.Body, idempotencyBodyMax+1))
			if err != nil {
				Write(w, r, NewRequestInvalid().
					WithDetail("The body of the request did not arrive in full."))

				return
			}

			if len(body) > idempotencyBodyMax {
				r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), r.Body))

				next.ServeHTTP(w, r)

				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))

			if done := replay(w, r, logger, store, key, digestOf(r, body)); done {
				return
			}

			recorder := &recordedAnswer{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)

			if recorder.status >= http.StatusBadRequest {
				return
			}

			// A store that does not keep the answer leaves the key free, so the
			// next repeat runs the write again. That is the safe direction, and
			// it is invisible without this line.
			if err = store.Keep(r.Context(), key, IdempotentAnswer{
				RequestHash: digestOf(r, body),
				Status:      recorder.status,
				Header:      recorder.header,
				Body:        recorder.body.Bytes(),
			}); err != nil {
				logger.ErrorContext(r.Context(), "the key cache kept no answer",
					slog.String("idempotency_key", key), slog.Any("error", err))
			}
		})
	}
}

// replay answers the stored result, and reports whether it did.
func replay(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	store IdempotencyStore,
	key string,
	hash string,
) bool {
	held, err := store.Answer(r.Context(), key)
	if err != nil {
		// An absent key is the ordinary case. Anything else means the cache
		// failed, and the write runs again rather than refusing the request.
		if !errors.Is(err, ErrNoIdempotentAnswer) {
			logger.ErrorContext(r.Context(), "the key cache answered nothing",
				slog.String("idempotency_key", key), slog.Any("error", err))
		}

		return false
	}

	// A header that this request already carries stays: the chain sets the flow
	// identifier and the cache period for this request, not for the first one.
	if held.RequestHash != hash {
		Write(w, r, NewRequestInvalid().WithDetail(
			"The Idempotency-Key header names an earlier request that differs from this one.",
		))

		return true
	}

	for name, values := range held.Header {
		if w.Header().Get(name) != "" {
			continue
		}

		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	w.WriteHeader(held.Status)
	_, _ = w.Write(held.Body)

	return true
}

// digestOf builds the request hash that the cache compares.
func digestOf(r *http.Request, body []byte) string {
	sum := sha256.New()
	sum.Write([]byte(r.Method))
	sum.Write([]byte{0})
	sum.Write([]byte(r.URL.Path))
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

func (a *recordedAnswer) Write(payload []byte) (int, error) {
	if !a.written {
		a.WriteHeader(http.StatusOK)
	}

	a.body.Write(payload)

	//nolint:wrapcheck // the wrapped writer owns the error of the write.
	return a.ResponseWriter.Write(payload)
}
