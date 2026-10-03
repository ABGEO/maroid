package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/abgeo/maroid/libs/rest/problem"
)

// unknownFieldPrefix starts the error that encoding/json answers for a member that
// the target does not declare.
const unknownFieldPrefix = "json: unknown field "

// decodeObject reads the body into target and refuses a member that target does not
// declare. It answers the problem to write, or nil when the body decoded.
func decodeObject(r *http.Request, target any) *problem.Problem {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(target)
	if err == nil {
		return nil
	}

	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) && strings.HasPrefix(err.Error(), unknownFieldPrefix) {
		return problem.NewMemberUnknown().WithDetail(
			"The body carries " + strings.TrimPrefix(err.Error(), unknownFieldPrefix) + ".",
		)
	}

	return problem.NewBodyInvalid()
}
