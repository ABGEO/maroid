package rest

// ProblemMediaType is the content type of every error response. The name
// carries the subject, because this module also answers application/json.
const ProblemMediaType = "application/problem+json"

// Problem is the body of an error response.
type Problem struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Detail   string         `json:"detail,omitempty"`
	Instance string         `json:"instance,omitempty"`
	Errors   []FieldFailure `json:"errors,omitempty"`
}

// FieldFailure names one field that caused a rejection.
type FieldFailure struct {
	Detail  string `json:"detail"`
	Pointer string `json:"pointer"`
}

// WithDetail returns the problem with the detail. A detail names this occurrence
// and never carries the text of an error, a query, or a secret.
func (p Problem) WithDetail(detail string) Problem {
	p.Detail = detail

	return p
}

// WithErrors returns the problem with one item for each field that failed.
func (p Problem) WithErrors(failures ...FieldFailure) Problem {
	p.Errors = failures

	return p
}

// WithStatus returns the problem with another status. A caller that holds a
// status and no cause uses it, such as the one that rewrites a failure of a
// transport it does not own.
func (p Problem) WithStatus(status int) Problem {
	p.Status = status

	return p
}

// newProblem builds a problem of one registered type.
func newProblem(problemType, title string, status int) Problem {
	return Problem{Type: problemType, Title: title, Status: status}
}
