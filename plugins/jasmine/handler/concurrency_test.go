package handler_test

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/jasmine/db"
	"github.com/abgeo/maroid/plugins/jasmine/handler"
)

const pluginID = "dev.maroid.jasmine"

// setUpdatedAt is the trigger function that a core migration of the hub creates.
// A plugin test owns no core migration, so it creates the function itself.
const setUpdatedAt = `
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;`

// routerUnderTest mounts the environment routes over a migrated database.
func routerUnderTest(t *testing.T) http.Handler {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(setUpdatedAt)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID(pluginID)
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	routes := handler.NewEnvironmentHandler(
		slog.New(slog.DiscardHandler),
		pluginapi.NewPluginDB(instance.DB, identifier),
	).Routes()

	router := chi.NewRouter()

	// The hub mounts this for every route, so a plugin route reads the validator
	// off the context and parses nothing.
	router.Use(precondition.IfMatch)

	for _, route := range routes {
		router.Method(route.Method, route.Pattern, route.Handler)
	}

	return router
}

// send runs one request and answers the recorder that holds its answer.
func send(
	t *testing.T,
	router http.Handler,
	method string,
	target string,
	body string,
	ifMatch string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(
		t.Context(), method, target, strings.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")

	if ifMatch != "" {
		request.Header.Set(precondition.IfMatchHeader, ifMatch)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

// nameOf reads the name that an answer carries.
func nameOf(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var answer struct {
		Name string `json:"name"`
	}

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))

	return answer.Name
}

// createEnvironment makes one record and answers its address with its validator.
func createEnvironment(t *testing.T, router http.Handler) (string, string) {
	t.Helper()

	recorder := send(t, router, http.MethodPost, "/environments", `{"name":"Balcony"}`, "")
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())

	var answer struct {
		ID string `json:"id"`
	}

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))

	tag := recorder.Header().Get(precondition.ETagHeader)
	require.NotEmpty(t, tag, "a write answers the validator of what it stored")

	return "/environments/" + answer.ID, tag
}

// APIFMT-SC-019: Two clients read one record and both write it back with the
// ETag that they read. The first write lands, and the second answers 412.
func TestTheSecondOfTwoWritesUnderOneValidatorIsRefused(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)
	address, _ := createEnvironment(t, router)

	read := send(t, router, http.MethodGet, address, "", "")
	require.Equal(t, http.StatusOK, read.Code)

	shared := read.Header().Get(precondition.ETagHeader)
	require.NotEmpty(t, shared, "a read answers the validator of the record")

	first := send(t, router, http.MethodPut, address, `{"name":"Shelf"}`, shared)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())

	second := send(t, router, http.MethodPut, address, `{"name":"Windowsill"}`, shared)
	assert.Equal(t, http.StatusPreconditionFailed, second.Code, second.Body.String())

	after := send(t, router, http.MethodGet, address, "", "")
	assert.Equal(t, "Shelf", nameOf(t, after), "the refused write changed nothing")
}

// APIFMT-SC-019: A write answers the validator of what it stored, so a client
// writes again without reading again.
func TestAWriteAnswersTheValidatorOfWhatItStored(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)
	address, created := createEnvironment(t, router)

	first := send(t, router, http.MethodPut, address, `{"name":"Shelf"}`, created)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())

	answered := first.Header().Get(precondition.ETagHeader)
	require.NotEmpty(t, answered)
	assert.NotEqual(t, created, answered, "a write moves the record, so the validator moves")

	second := send(t, router, http.MethodPut, address, `{"name":"Windowsill"}`, answered)
	assert.Equal(t, http.StatusOK, second.Code, second.Body.String())
}

// APIFMT-SC-019, the limit case: a write that names no validator lands, because
// APIFMT-DD-014 keeps the header optional.
func TestAWriteThatNamesNoValidatorLands(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)
	address, _ := createEnvironment(t, router)

	written := send(t, router, http.MethodPut, address, `{"name":"Shelf"}`, "")
	require.Equal(t, http.StatusOK, written.Code, written.Body.String())
	assert.Equal(t, "Shelf", nameOf(t, written))
}

// APIFMT-SC-019: a validator that this API never answered is a bad request, not
// a failed precondition, because the client built a value of its own.
func TestAWriteUnderAValidatorThatMaroidDidNotAnswerIsRefused(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)
	address, _ := createEnvironment(t, router)

	written := send(t, router, http.MethodPut, address, `{"name":"Shelf"}`, `"not-a-moment"`)
	assert.Equal(t, http.StatusBadRequest, written.Code, written.Body.String())
}
