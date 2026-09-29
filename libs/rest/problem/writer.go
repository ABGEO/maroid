package problem

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/abgeo/maroid/libs/rest/flow"
)

// fallbackBody answers when a problem does not encode. It is a constant, so it
// needs no encoder of its own and cannot fail in turn.
const fallbackBody = `{"type":"` + TypeInternal +
	`","title":"The request failed.","status":500}`

// Fill sets the members that belong on the wire for this request. It takes the
// instance from the request, and it drops the detail of an internal problem,
// because the cause of one belongs in the log and not in a body.
//
// Write calls it. A caller that writes the body itself calls it first.
func Fill(r *http.Request, body Body) {
	prob := body.Base()

	if prob.Type == TypeInternal {
		prob.Detail = ""
	}

	if prob.Instance == "" {
		prob.Instance = flow.Instance(flow.IDFromContext(r.Context()))
	}
}

// Write answers the request with the problem.
//
// It encodes before it writes the status, so a body that does not encode still
// leaves the answer well formed. It reports nothing, because the one failure
// that remains is a peer that has gone, and the answer has left by then. The
// access record of that request carries the status and the bytes that arrived.
func Write(w http.ResponseWriter, r *http.Request, body Body) {
	Fill(r, body)

	status := body.Base().Status

	var encoded bytes.Buffer
	if err := json.NewEncoder(&encoded).Encode(body); err != nil {
		encoded.Reset()
		encoded.WriteString(fallbackBody)

		status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", MediaType)
	w.WriteHeader(status)

	_, _ = io.Copy(w, &encoded)
}
