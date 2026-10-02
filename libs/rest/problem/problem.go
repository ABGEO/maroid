package problem

// MediaType is the content type of every error response. The name
// carries the subject, because this module also answers application/json.
const MediaType = "application/problem+json"

// Body is a problem, or a type that embeds one and adds members of its own.
// Write answers any of them.
type Body interface {
	Base() *Problem
}

// Problem is the body of an error response.
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// ValidationProblem is a problem that names each field that caused a rejection.
type ValidationProblem struct {
	Problem

	Errors []FieldFailure `json:"errors"`
}

// PermissionDeniedProblem is a problem that names the permission that an action
// needs, and the lowest role that holds it.
type PermissionDeniedProblem struct {
	Problem

	Permission   string `json:"permission"`
	RequiredRole string `json:"required_role,omitempty"`
}

// FieldFailure names one field that caused a rejection.
type FieldFailure struct {
	Detail  string `json:"detail"`
	Pointer string `json:"pointer"`
}

var (
	_ Body = (*Problem)(nil)
	_ Body = (*ValidationProblem)(nil)
	_ Body = (*PermissionDeniedProblem)(nil)
)

// Base returns the problem itself. A type that embeds a problem gets it through
// the embedded field.
func (p *Problem) Base() *Problem {
	return p
}

// WithDetail sets the detail and returns the problem. A detail names this
// occurrence and never carries the text of an error, a query, or a secret.
func (p *Problem) WithDetail(detail string) *Problem {
	p.Detail = detail

	return p
}

// WithStatus sets another status and returns the problem. A caller that holds a
// status and no cause uses it, such as the one that rewrites a failure of a
// transport it does not own.
func (p *Problem) WithStatus(status int) *Problem {
	p.Status = status

	return p
}

// newProblem builds a problem of one registered type.
func newProblem(problemType, title string, status int) *Problem {
	return &Problem{Type: problemType, Title: title, Status: status}
}
