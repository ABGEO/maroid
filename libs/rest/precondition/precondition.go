// Package precondition carries the entity tag of a record and the conditional
// write that it guards.
package precondition

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/abgeo/maroid/libs/rest/problem"
)

const (
	// IfMatchHeader carries the validator that a client read, on a write that must
	// not replace a record which moved since.
	IfMatchHeader = "If-Match"
	// ETagHeader carries the validator of a record whose last write is at moment.
	ETagHeader = "ETag"
)

type contextKey int

const ifMatchKey contextKey = 0

var (
	// ErrModified reports a write whose record moved after the client read it. A
	// repository answers it when the conditional update changes no row, and the
	// handler turns it into a 412.
	ErrModified = errors.New("precondition: record changed after the read")

	errValidatorInvalid = errors.New("precondition: validator not answered by this API")
)

// ETag returns the validator of a record whose last write is at moment. The
// value is strong, because it names one version of the stored record, and RFC
// 9110 lets only a strong validator guard a write.
func ETag(moment time.Time) string {
	return `"` + strconv.FormatInt(moment.UnixNano(), 10) + `"`
}

// IfMatch reads the validator of a conditional write once, and the context of
// every handler behind it carries the moment. A value that this API never
// answered ends the request here, so no handler reads a moment it cannot trust,
// and no route repeats the parse.
//
// A request that names no validator reaches its handler with a nil moment, and
// such a write lands, because the guideline asks for the optimistic path and
// does not make the header mandatory.
func IfMatch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		moment, held, err := readIfMatch(r)
		if err != nil {
			problem.Write(w, r, problem.NewRequestInvalid().
				WithDetail("The If-Match header is not a validator that this API answered."))

			return
		}

		if !held {
			next.ServeHTTP(w, r)

			return
		}

		next.ServeHTTP(w, r.WithContext(withIfMatch(r.Context(), &moment)))
	})
}

// withIfMatch returns a context that carries the moment of a conditional
// write. A nil moment is the write that names no validator.
func withIfMatch(ctx context.Context, moment *time.Time) context.Context {
	return context.WithValue(ctx, ifMatchKey, moment)
}

// IfMatchFromContext returns the moment that the write must match, or nil when
// the request named none and the write lands unconditionally.
func IfMatchFromContext(ctx context.Context) *time.Time {
	moment, _ := ctx.Value(ifMatchKey).(*time.Time)

	return moment
}

// readIfMatch reads the validator that a write carries. The second answer is
// false when the request carries none.
//
// The value `*` is refused. RFC 9110 gives it the meaning that the record must
// exist, and a write to a record that does not exist already answers a 404, so
// nothing of Maroid needs it.
func readIfMatch(r *http.Request) (time.Time, bool, error) {
	raw := strings.TrimSpace(r.Header.Get(IfMatchHeader))
	if raw == "" {
		return time.Time{}, false, nil
	}

	moment, err := validatorMoment(raw)
	if err != nil {
		return time.Time{}, false, err
	}

	return moment, true, nil
}

// validatorMoment reads the moment out of one strong validator.
func validatorMoment(validator string) (time.Time, error) {
	if len(validator) < 2 || validator[0] != '"' || validator[len(validator)-1] != '"' {
		return time.Time{}, fmt.Errorf("reading %q: %w", validator, errValidatorInvalid)
	}

	nanoseconds, err := strconv.ParseInt(validator[1:len(validator)-1], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("reading %q: %w", validator, errValidatorInvalid)
	}

	return time.Unix(0, nanoseconds).UTC(), nil
}
