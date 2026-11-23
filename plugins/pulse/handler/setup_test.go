package handler_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/plugins/pulse/handler"
	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
)

const (
	nina = pulsetest.Nina
	beka = pulsetest.Beka

	externalURL = "https://maroid.example"
)

// api is pulse behind its routes, as the hub mounts them.
type api struct {
	*pulsetest.Pulse
}

func newAPI(t *testing.T) *api {
	t.Helper()

	return &api{Pulse: pulsetest.New(t)}
}

// router mounts the routes with the user as the acting user, in the zone of the user.
func (app *api) router(user string) http.Handler {
	logger := slog.New(slog.DiscardHandler)

	routes := slices.Concat(
		handler.NewSettingsHandler(logger, app.Settings, app.Clock.Now).Routes(),
		handler.NewQuestionHandler(logger, app.DB, app.Clock.Now).Routes(),
		handler.NewValueHandler(logger, app.DB).Routes(),
		handler.NewCheckinHandler(logger, app.DB, app.Clock.Now).Routes(),
		handler.NewRoundHandler(logger, app.DB).Routes(),
	)

	router := chi.NewRouter()
	router.Use(address.Middleware(externalURL))
	router.Use(precondition.IfMatch)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			acting := pulsetest.ActingAs(user)
			ctx := pluginapi.ContextWithActingTimeZone(
				pluginapi.ContextWithActingUser(
					r.Context(),
					user,
				),
				pluginapi.ActingTimeZone(acting),
			)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	for _, route := range routes {
		router.Method(route.Method, route.Pattern, route.Handler)
	}

	return router
}

// send runs one request as the user and answers the recorder.
func (app *api) send(
	t *testing.T,
	user, method, target, body, ifMatch string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	if ifMatch != "" {
		request.Header.Set(precondition.IfMatchHeader, ifMatch)
	}

	recorder := httptest.NewRecorder()
	app.router(user).ServeHTTP(recorder, request)

	return recorder
}

// as runs one request as Nina with no validator.
func (app *api) as(t *testing.T, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	return app.send(t, nina, method, target, body, "")
}

func decode[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	var body T
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body), recorder.Body.String())

	return body
}

type failure struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
	Errors []struct {
		Pointer string `json:"pointer"`
	} `json:"errors"`
}

// pointers answers the pointers of a 422.
func pointers(t *testing.T, recorder *httptest.ResponseRecorder) []string {
	t.Helper()

	body := decode[failure](t, recorder)

	out := make([]string, 0, len(body.Errors))
	for _, item := range body.Errors {
		out = append(out, item.Pointer)
	}

	return out
}

type question struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Prompt     string          `json:"prompt"`
	Archived   bool            `json:"archived"`
	Config     json.RawMessage `json:"config"`
	ValueCount int             `json:"value_count"`
	Latest     *struct {
		Number *float64 `json:"number"`
	} `json:"latest"`
}

type page[T any] struct {
	Items []T     `json:"items"`
	Next  *string `json:"next"`
}

// insertValue stores a value of the question as Nina, with the number at the moment.
func (app *api) insertNumber(
	t *testing.T,
	questionID string,
	kind string,
	moment string,
	number float64,
) {
	t.Helper()

	pulsetest.Exec(t, app.DB, nina, `
		INSERT INTO question_values (question_id, type, occurred_at, answered_at, source, number_value)
		VALUES ($1, $2, $3, $3, 'telegram', $4)`, questionID, kind, moment, number)
}
