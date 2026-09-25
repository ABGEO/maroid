package rest

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// IfMatchHeader carries the validator that a client read, on a write that must
// not replace a record which moved since.
const IfMatchHeader = "If-Match"

var (
	// ErrModified reports a write whose record moved after the client read it. A
	// repository answers it when the conditional update changes no row, and the
	// handler turns it into a 412.
	ErrModified = errors.New("the record changed after the client read it")

	errValidatorInvalid = errors.New("the validator is not one that this API answered")
)

// ETag returns the validator of a record whose last write is at moment. The
// value is weak, because two answers with one moment can still differ in a
// member that another table feeds.
func ETag(moment time.Time) string {
	return `W/"` + strconv.FormatInt(moment.UnixNano(), 10) + `"`
}

// IfMatch reads the validator that a write carries. The second answer is false
// when the request carries none, and such a write lands, because the guideline
// asks for the optimistic path and does not make the header mandatory.
//
// The value `*` is refused. RFC 9110 gives it the meaning that the record must
// exist, and a write to a record that does not exist already answers a 404, so
// nothing of Maroid needs it.
func IfMatch(r *http.Request) (time.Time, bool, error) {
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

// validatorMoment reads the moment out of one validator, weak or strong.
func validatorMoment(validator string) (time.Time, error) {
	quoted := strings.TrimPrefix(validator, "W/")

	if len(quoted) < 2 || quoted[0] != '"' || quoted[len(quoted)-1] != '"' {
		return time.Time{}, fmt.Errorf("reading %q: %w", validator, errValidatorInvalid)
	}

	nanoseconds, err := strconv.ParseInt(quoted[1:len(quoted)-1], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("reading %q: %w", validator, errValidatorInvalid)
	}

	return time.Unix(0, nanoseconds).UTC(), nil
}
