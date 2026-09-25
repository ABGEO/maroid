package rest

import "net/http"

// The problem types that any component of Maroid answers. Each one names a
// failure of the protocol and carries no fact of a domain, so the hub and a
// plugin both reach for it.
const (
	TypeRequestInvalid   = "/problems/http/request-invalid"
	TypeBodyInvalid      = "/problems/http/body-invalid"
	TypeAccessDenied     = "/problems/http/access-denied"
	TypeNotFound         = "/problems/http/not-found"
	TypeMethodNotAllowed = "/problems/http/method-not-allowed"
	TypeValidationFailed = "/problems/http/validation-failed"
	TypeInternal         = "/problems/http/internal"
)

// NewRequestInvalid reports a request that a route cannot read.
func NewRequestInvalid() Problem {
	return newProblem(TypeRequestInvalid, "The request is not valid.", http.StatusBadRequest)
}

// NewBodyInvalid reports a body that does not decode as a JSON object.
func NewBodyInvalid() Problem {
	return newProblem(TypeBodyInvalid, "The body is not a JSON object.", http.StatusBadRequest)
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
