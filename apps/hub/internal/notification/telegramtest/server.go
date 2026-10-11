// Package telegramtest gives a stub of the Telegram Bot API that records each call.
package telegramtest

import (
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const maxFormMemory = 32 << 20

// Call is one request that the stub received.
type Call struct {
	Method string
	Params map[string]string
	Files  []string
	At     time.Time
}

// Server is a stub of the Bot API. It answers every call with ok: true unless a
// refusal or a silence names the chat of the call.
type Server struct {
	server *httptest.Server

	mu       sync.Mutex
	calls    []Call
	refusals map[string]string
	silent   bool
}

// NewServer starts a stub that the test closes at its end.
func NewServer(t *testing.T) *Server {
	t.Helper()

	stub := &Server{refusals: make(map[string]string)}
	stub.server = httptest.NewServer(http.HandlerFunc(stub.serve))
	t.Cleanup(stub.server.Close)

	return stub
}

// URL is the address that telego.WithAPIServer takes.
func (s *Server) URL() string {
	return s.server.URL
}

// Refuse makes every call that names the chat answer ok: false with the description.
// The chat "" names a call that carries no chat, such as getMe.
func (s *Server) Refuse(chatID string, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.refusals[chatID] = description
}

// Silence makes every call wait until the client gives up.
func (s *Server) Silence() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.silent = true
}

// Calls returns every call, in the order that the stub received them.
func (s *Server) Calls() []Call {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]Call(nil), s.calls...)
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	call := Call{Method: segments[len(segments)-1], Params: map[string]string{}, At: time.Now()}

	readParams(r, &call)

	s.mu.Lock()
	s.calls = append(s.calls, call)
	refusal, refused := s.refusals[call.Params["chat_id"]]
	silent := s.silent
	s.mu.Unlock()

	if silent {
		<-r.Context().Done()

		return
	}

	w.Header().Set("Content-Type", "application/json")

	if refused {
		writeJSON(w, map[string]any{
			"ok": false, "error_code": http.StatusBadRequest, "description": refusal,
		})

		return
	}

	writeJSON(w, map[string]any{"ok": true, "result": resultOf(call)})
}

func writeJSON(w http.ResponseWriter, body map[string]any) {
	if err := json.NewEncoder(w).Encode(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func readParams(r *http.Request, call *Call) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxFormMemory)

	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))

	if mediaType != "multipart/form-data" {
		raw := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&raw)

		for key, value := range raw {
			call.Params[key] = stringOf(value)
		}

		return
	}

	//nolint:gosec // MaxBytesReader at the start of readParams bounds the body.
	if err := r.ParseMultipartForm(maxFormMemory); err != nil {
		return
	}

	for key, values := range r.MultipartForm.Value {
		call.Params[key] = values[0]
	}

	for _, headers := range r.MultipartForm.File {
		for _, header := range headers {
			call.Files = append(call.Files, header.Filename)
		}
	}
}

func stringOf(value any) string {
	if text, ok := value.(string); ok {
		return text
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}

	return string(encoded)
}

func resultOf(call Call) any {
	message := map[string]any{
		"message_id": 1,
		"date":       0,
		"chat":       map[string]any{"id": 1, "type": "group"},
	}

	switch call.Method {
	case "getMe":
		return map[string]any{"id": 1, "is_bot": true, "first_name": "Maroid"}
	case "sendMediaGroup":
		return []any{message}
	default:
		return message
	}
}
