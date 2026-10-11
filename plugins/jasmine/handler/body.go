package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// unknownFieldPrefix starts the error that encoding/json answers for a member that
// the target does not declare.
const unknownFieldPrefix = "json: unknown field "

const (
	// paramID names the identifier segment of a route of one record.
	paramID = "id"

	// canonicalUUIDLength is the length of a UUID in the form 8-4-4-4-12.
	canonicalUUIDLength = 36
)

// validatable is a request body that checks its own rules after it decodes.
type validatable interface {
	Validate() error
}

// readBody decodes the body into target and checks its rules. A member that
// target does not declare is refused rather than dropped, so a client learns that
// it sent a value that no write takes. It writes the problem and answers false
// when the body is refused.
func readBody(w http.ResponseWriter, r *http.Request, target validatable) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(target)

	var syntaxErr *json.SyntaxError

	switch {
	case err == nil:
	case !errors.As(err, &syntaxErr) && strings.HasPrefix(err.Error(), unknownFieldPrefix):
		problem.Write(w, r, problem.NewMemberUnknown().WithDetail(
			"The body carries "+strings.TrimPrefix(err.Error(), unknownFieldPrefix)+".",
		))

		return false
	default:
		problem.Write(w, r, problem.NewBodyInvalid())

		return false
	}

	if err = target.Validate(); err != nil {
		writeRequestProblem(w, r, err)

		return false
	}

	return true
}

// pathID reads the identifier segment of the route that name names. A value that
// is no UUID names no record, so it answers 404 before any query runs.
func pathID(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	id := chi.URLParam(r, name)
	if !isUUID(id) {
		problem.Write(w, r, problem.NewNotFound())

		return "", false
	}

	return id, true
}

// isUUID reports whether value is a UUID in the canonical form 8-4-4-4-12.
// uuid.Parse also takes the forms with braces and with the urn:uuid: prefix,
// which PostgreSQL refuses, so the length check keeps them out.
func isUUID(value string) bool {
	_, err := uuid.Parse(value)

	return err == nil && len(value) == canonicalUUIDLength
}

// location answers the address of the record that a create made, below the
// collection that the request named.
func location(r *http.Request, id string) string {
	return address.BaseFromContext(r.Context()) + r.URL.EscapedPath() + "/" + id
}
