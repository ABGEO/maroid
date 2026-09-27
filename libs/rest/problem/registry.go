package problem

import "net/http"

// The problem types that any component of Maroid answers. Each one names a
// failure of the protocol and carries no fact of a domain, so the hub and a
// plugin both reach for it.
const (
	TypeRequestInvalid     = "/problems/http/request-invalid"
	TypeBodyInvalid        = "/problems/http/body-invalid"
	TypeMemberUnknown      = "/problems/http/member-unknown"
	TypeAccessDenied       = "/problems/http/access-denied"
	TypeCursorStale        = "/problems/http/cursor-stale"
	TypeNotFound           = "/problems/http/not-found"
	TypeMethodNotAllowed   = "/problems/http/method-not-allowed"
	TypeRequestInProgress  = "/problems/http/request-in-progress"
	TypePreconditionFailed = "/problems/http/precondition-failed"
	TypeContentTooLarge    = "/problems/http/content-too-large"
	TypeValidationFailed   = "/problems/http/validation-failed"
	TypeInternal           = "/problems/http/internal"
)

// NewRequestInvalid reports a request that a route cannot read.
func NewRequestInvalid() Problem {
	return newProblem(TypeRequestInvalid, "The request is not valid.", http.StatusBadRequest)
}

// NewBodyInvalid reports a body that does not decode as a JSON object.
func NewBodyInvalid() Problem {
	return newProblem(TypeBodyInvalid, "The body is not a JSON object.", http.StatusBadRequest)
}

// NewMemberUnknown reports a body that carries a member which the schema of the
// route does not declare. A route refuses the member rather than drop it, so a
// client learns of a misspelled member.
func NewMemberUnknown() Problem {
	return newProblem(
		TypeMemberUnknown,
		"The body carries a member that the schema does not declare.",
		http.StatusBadRequest,
	)
}

// NewCursorStale reports a cursor whose sort or filters differ from the request
// that carries it. A client tells it from a cursor it built wrong, and retires
// the one it holds. RES-006.
func NewCursorStale() Problem {
	return newProblem(
		TypeCursorStale,
		"The cursor does not match this request.",
		http.StatusBadRequest,
	)
}

// NewAccessDenied reports a request that carries no active user record. Every
// condition of a 401 answers with this one, so a caller learns nothing about
// which account exists.
func NewAccessDenied() Problem {
	return newProblem(
		TypeAccessDenied,
		"The request carries no active user record.",
		http.StatusUnauthorized,
	)
}

// NewNotFound reports a resource that does not exist. A resource of another user
// answers the same way, because the row level policy hides it.
func NewNotFound() Problem {
	return newProblem(TypeNotFound, "The resource does not exist.", http.StatusNotFound)
}

// NewMethodNotAllowed reports a method that does not reach the resource.
func NewMethodNotAllowed() Problem {
	return newProblem(
		TypeMethodNotAllowed,
		"The method does not reach this resource.",
		http.StatusMethodNotAllowed,
	)
}

// NewRequestInProgress reports a repeat that arrived while the first write under
// its key still runs. The client sends it again later and reads the answer of
// that first write.
func NewRequestInProgress() Problem {
	return newProblem(
		TypeRequestInProgress,
		"An earlier request under this key is still running.",
		http.StatusConflict,
	)
}

// NewPreconditionFailed reports a write whose record moved after the client read
// it. The client reads the record again and writes what it still means to write.
func NewPreconditionFailed() Problem {
	return newProblem(
		TypePreconditionFailed,
		"The record changed after the client read it.",
		http.StatusPreconditionFailed,
	)
}

// NewContentTooLarge reports a body that is larger than the route takes.
func NewContentTooLarge() Problem {
	return newProblem(
		TypeContentTooLarge,
		"The body is larger than the route takes.",
		http.StatusRequestEntityTooLarge,
	)
}

// NewValidationFailed reports a body that the rules of a route refuse.
func NewValidationFailed() Problem {
	return newProblem(
		TypeValidationFailed,
		"The body failed validation.",
		http.StatusUnprocessableEntity,
	)
}

// NewInternal reports a failure that Maroid does not describe to the caller.
// The cause belongs in the log, so this problem carries no detail.
func NewInternal() Problem {
	return newProblem(TypeInternal, "The request failed.", http.StatusInternalServerError)
}
