package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
)

const (
	paramID = "id"

	// unknownFieldPrefix starts the error that encoding/json answers for a member
	// that the target does not declare.
	unknownFieldPrefix = "json: unknown field "

	// canonicalUUIDLength is the length of a UUID in the form 8-4-4-4-12.
	canonicalUUIDLength = 36

	bodyMax = 64 << 10
)

// validatable is a request body that checks its own limits after it decodes.
type validatable interface {
	validate() []problem.FieldFailure
}

// readBody decodes the body into target and checks its limits. It writes the
// problem and answers false when the body is refused.
func readBody(w http.ResponseWriter, r *http.Request, target validatable) bool {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, bodyMax))

	var tooLarge *http.MaxBytesError

	switch {
	case errors.As(err, &tooLarge):
		problem.Write(
			w,
			r,
			problem.NewContentTooLarge().WithDetail("The body holds at most 64 KiB."),
		)

		return false
	case err != nil:
		problem.Write(w, r, problem.NewBodyInvalid())

		return false
	}

	if !decodeStrict(w, r, raw, target) {
		return false
	}

	if failures := target.validate(); len(failures) > 0 {
		problem.Write(w, r, problem.NewValidationFailed(failures...))

		return false
	}

	return true
}

// decodeStrict decodes raw into target. A member that target does not declare is
// refused rather than dropped, so a client learns that no write takes it.
func decodeStrict(w http.ResponseWriter, r *http.Request, raw []byte, target any) bool {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	err := decoder.Decode(target)

	var syntaxErr *json.SyntaxError

	switch {
	case err == nil:
		return true
	case !errors.As(err, &syntaxErr) && strings.HasPrefix(err.Error(), unknownFieldPrefix):
		problem.Write(w, r, problem.NewMemberUnknown().WithDetail(
			"The body carries "+strings.TrimPrefix(err.Error(), unknownFieldPrefix)+".",
		))
	default:
		problem.Write(w, r, problem.NewBodyInvalid())
	}

	return false
}

// pathID reads the identifier segment of the route. A value that is no UUID names no
// record, so it answers 404 before any query runs.
func pathID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := chi.URLParam(r, paramID)
	if !isUUID(id) {
		problem.Write(w, r, problem.NewNotFound())

		return "", false
	}

	return id, true
}

// isUUID reports whether value is a UUID in the canonical form 8-4-4-4-12.
func isUUID(value string) bool {
	_, err := uuid.Parse(value)

	return err == nil && len(value) == canonicalUUIDLength
}

// location answers the address of the record that a create made, below the
// collection that the request named.
func location(r *http.Request, id string) string {
	return address.BaseFromContext(r.Context()) + r.URL.EscapedPath() + "/" + id
}

// answer writes the body with the validator of the moment.
func answer(w http.ResponseWriter, r *http.Request, status int, moment time.Time, body any) {
	w.Header().Set(precondition.ETagHeader, precondition.ETag(moment))
	render.Status(r, status)
	render.JSON(w, r, body)
}

// inTx runs work inside a transaction.
func inTx(
	ctx context.Context,
	db pluginapi.PluginDB,
	action string,
	work func(ctx context.Context, tx *sqlx.Tx) error,
) error {
	if err := db.WithTx(ctx, func(tx *sqlx.Tx) error { return work(ctx, tx) }); err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	return nil
}

// fieldFailures collects the members of a body that break a limit.
type fieldFailures []problem.FieldFailure

func (f *fieldFailures) add(pointer string, detail string) {
	*f = append(*f, problem.FieldFailure{Pointer: "#/" + pointer, Detail: detail})
}

// length checks that a text holds 1 to high characters.
func (f *fieldFailures) length(pointer string, value string, high int) {
	if count := len([]rune(value)); count < 1 || count > high {
		f.add(pointer, fmt.Sprintf("must hold 1 to %d characters", high))
	}
}
