package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/dex/dextest"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
)

const (
	ninaEmail       = "nina@home.example"
	ninaPassword    = "correct-horse-1"
	ninaNewPassword = "battery-staple-2"
	ninaAccount     = "111"
	bekaAccount     = "222"
	ownPasswordPath = "/auth/identities/" + auth.ProviderLocal
)

// profileUnderTest holds Nina, with a local account and a Telegram identity, and Beka,
// with a Telegram identity only.
func profileUnderTest(t *testing.T) *authFixture {
	t.Helper()

	fixture := authUnderTest(t)
	ctx := t.Context()

	require.NoError(t, fixture.idp.CreateConnector(ctx,
		dextest.Connector(auth.ProviderLocal, "local", "Email", `{"maroidPreset":"local"}`)))

	nina := addUserRecord(t, fixture.database, "Nina")
	require.NoError(t, fixture.service.Attach(
		ctx, nina, auth.ProviderTelegram, ninaAccount, model.Profile{},
	))
	require.NoError(t, fixture.accounts.Give(ctx, nina, ninaEmail, []byte(ninaPassword)))

	beka := addUserRecord(t, fixture.database, "Beka")
	require.NoError(t, fixture.service.Attach(
		ctx, beka, auth.ProviderTelegram, bekaAccount, model.Profile{},
	))

	return fixture
}

func (f *authFixture) changeOwnPassword(
	t *testing.T,
	accountID string,
	path string,
	body map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()

	encoded, err := json.Marshal(body)
	require.NoError(t, err)

	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodPatch, path, bytes.NewReader(encoded),
	)
	request.Header.Set("Content-Type", "application/merge-patch+json")
	request.AddCookie(
		requestCookie(sessionCookie, f.provider.Sign(t, auth.ProviderTelegram, accountID)),
	)

	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, request)

	return recorder
}

func (f *authFixture) verifies(t *testing.T, password string) bool {
	t.Helper()

	verified, err := f.idp.VerifyPassword(t.Context(), ninaEmail, []byte(password))
	require.NoError(t, err)

	return verified
}

func ownPasswordBody(current string, password string) map[string]any {
	return map[string]any{"current_password": current, "password": password}
}

func requireFieldFailure(t *testing.T, recorder *httptest.ResponseRecorder, pointer string) {
	t.Helper()

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())

	var failure problem.ValidationProblem

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &failure))
	require.Len(t, failure.Errors, 1)
	assert.Equal(t, pointer, failure.Errors[0].Pointer)
}

// PROFILE-SC-006: Nina sets a new password with the current one. Dex holds the new
// one, and her session stays.
func TestAPersonChangesTheirOwnPassword(t *testing.T) {
	t.Parallel()

	fixture := profileUnderTest(t)

	changed := fixture.changeOwnPassword(t, ninaAccount, ownPasswordPath,
		ownPasswordBody(ninaPassword, ninaNewPassword))
	require.Equal(t, http.StatusNoContent, changed.Code, changed.Body.String())

	assert.True(t, fixture.verifies(t, ninaNewPassword))
	assert.False(t, fixture.verifies(t, ninaPassword))
	assert.Equal(t, "Nina", fixture.me(t, auth.ProviderTelegram, ninaAccount)["first_name"])
}

// PROFILE-SC-007: A person with no local account, and a provider other than local,
// answer not-found. Dex keeps the password of Nina.
func TestAChangeOfAPasswordThatIsNotThereIsNotFound(t *testing.T) {
	t.Parallel()

	fixture := profileUnderTest(t)
	hash := fixture.idp.Hash(ninaEmail)

	beka := fixture.changeOwnPassword(t, bekaAccount, ownPasswordPath,
		ownPasswordBody(ninaPassword, ninaNewPassword))
	require.Equal(t, http.StatusNotFound, beka.Code, beka.Body.String())

	telegram := fixture.changeOwnPassword(t, ninaAccount,
		"/auth/identities/"+auth.ProviderTelegram,
		ownPasswordBody(ninaPassword, ninaNewPassword))
	require.Equal(t, http.StatusNotFound, telegram.Code, telegram.Body.String())

	assert.Equal(t, hash, fixture.idp.Hash(ninaEmail))
}

// PROFILE-SC-008: A wrong current password, and no current password, name
// /current_password. Dex keeps the old password.
func TestAChangeOfAPasswordNeedsTheCurrentOne(t *testing.T) {
	t.Parallel()

	fixture := profileUnderTest(t)

	requireFieldFailure(t, fixture.changeOwnPassword(t, ninaAccount, ownPasswordPath,
		ownPasswordBody("wrong-horse-1", ninaNewPassword)), "/current_password")
	requireFieldFailure(t, fixture.changeOwnPassword(t, ninaAccount, ownPasswordPath,
		map[string]any{"password": ninaNewPassword}), "/current_password")

	assert.True(t, fixture.verifies(t, ninaPassword))
}

// PROFILE-SC-010: A new password of 11 characters, and one of 73 bytes, name /password.
// Dex receives no call, so a Dex that fails changes no answer.
func TestANewPasswordOutOfLengthIsRefused(t *testing.T) {
	t.Parallel()

	fixture := profileUnderTest(t)
	fixture.idp.Fail(errs.ErrIDPUnavailable)

	requireFieldFailure(t, fixture.changeOwnPassword(t, ninaAccount, ownPasswordPath,
		ownPasswordBody(ninaPassword, "elevenchars")), "/password")
	requireFieldFailure(t, fixture.changeOwnPassword(t, ninaAccount, ownPasswordPath,
		ownPasswordBody(ninaPassword, strings.Repeat("a", 73))), "/password")
}

// PROFILE-SC-011: A Dex that passes its deadline at the update answers not-ready with
// the dependency dex, and Dex keeps the old password.
func TestAChangeOfAPasswordAnswersNotReadyWithoutDex(t *testing.T) {
	t.Parallel()

	fixture := profileUnderTest(t)
	fixture.idp.FailOnce("UpdatePassword", errs.ErrIDPUnavailable)

	failed := fixture.changeOwnPassword(t, ninaAccount, ownPasswordPath,
		ownPasswordBody(ninaPassword, ninaNewPassword))
	require.Equal(t, http.StatusServiceUnavailable, failed.Code, failed.Body.String())
	assert.Contains(t, failed.Body.String(), "dex")

	assert.True(t, fixture.verifies(t, ninaPassword))
}

const selfPath = "/users/self"

func (f *workspaceFixture) countOf(t *testing.T, query string, userID string) int {
	t.Helper()

	var found int

	require.NoError(t, f.database.Get(&found, query, userID))

	return found
}

func (f *workspaceFixture) firstNameOf(t *testing.T, userID string) string {
	t.Helper()

	var name string

	require.NoError(t, f.database.Get(&name,
		`SELECT coalesce(first_name, '') FROM public.users WHERE id = $1;`, userID))

	return name
}

// PROFILE-SC-001: A person who is no administrator reads their own record, with its
// version.
func TestAPersonReadsTheirOwnRecord(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	read := fixture.call(t, fixture.nino, http.MethodGet, selfPath, nil, "")
	require.Equal(t, http.StatusOK, read.Code, read.Body.String())
	assert.NotEmpty(t, read.Header().Get(precondition.ETagHeader))

	body := decode(t, read)
	assert.Equal(t, fixture.nino.id, body["id"])
	assert.Equal(t, "Nino", body[firstNameK])
	assert.Equal(t, false, body["is_administrator"])
}

// PROFILE-SC-002: A person corrects their first name, then clears their last name. The
// identities, the workspaces, and the allowlist stay.
func TestAPersonChangesTheirOwnNames(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	beka := fixture.beka.id

	_, err := fixture.database.ExecContext(t.Context(),
		`UPDATE public.users SET first_name = 'Bkea', last_name = 'Kapanadze' WHERE id = $1;`,
		beka)
	require.NoError(t, err)

	queries := []string{
		`SELECT count(*) FROM public.identities WHERE user_id = $1;`,
		`SELECT count(*) FROM public.workspace_members WHERE user_id = $1;`,
		`SELECT count(*) FROM public.allowed_plugins WHERE user_id = $1;`,
	}
	before := make([]int, len(queries))

	for index, query := range queries {
		before[index] = fixture.countOf(t, query, beka)
	}

	etag := fixture.call(t, fixture.beka, http.MethodGet, selfPath, nil, "").
		Header().Get(precondition.ETagHeader)

	renamed := fixture.call(t, fixture.beka, http.MethodPatch, selfPath,
		map[string]any{firstNameK: "Beka"}, etag)
	require.Equal(t, http.StatusOK, renamed.Code, renamed.Body.String())
	assert.Equal(t, "Beka", decode(t, renamed)[firstNameK])
	assert.Equal(t, "Kapanadze", decode(t, renamed)[lastNameK])

	cleared := fixture.call(t, fixture.beka, http.MethodPatch, selfPath,
		map[string]any{lastNameK: ""}, "")
	require.Equal(t, http.StatusOK, cleared.Code, cleared.Body.String())
	assert.NotContains(t, decode(t, cleared), lastNameK, "a cleared name is absent")

	for index, query := range queries {
		assert.Equal(t, before[index], fixture.countOf(t, query, beka), query)
	}
}

// PROFILE-SC-003: A person who is no administrator cannot rename another record
// through the route of an administrator.
func TestAPersonCannotChangeTheNamesOfAnotherRecord(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	refused := fixture.call(t, fixture.nino, http.MethodPatch, "/users/"+fixture.beka.id,
		map[string]any{firstNameK: "X"}, "")
	require.Equal(t, http.StatusForbidden, refused.Code, refused.Body.String())
	assert.Equal(t, "Beka", fixture.firstNameOf(t, fixture.beka.id))
}

// PROFILE-SC-004: A change of the own record that names a field other than the names
// answers member-unknown, and no field changes.
func TestAPersonCannotChangeOtherFieldsOfTheirRecord(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	for _, body := range []map[string]any{
		{firstNameK: "Nina", "is_administrator": true},
		{"status": "active"},
	} {
		refused := fixture.call(t, fixture.nino, http.MethodPatch, selfPath, body, "")
		require.Equal(t, http.StatusBadRequest, refused.Code, refused.Body.String())
		assert.Equal(t, problem.TypeMemberUnknown, decode(t, refused)["type"])
	}

	read := decode(t, fixture.call(t, fixture.nino, http.MethodGet, selfPath, nil, ""))
	assert.Equal(t, "Nino", read[firstNameK])
	assert.Equal(t, false, read["is_administrator"])
}

// PROFILE-SC-012: Dex sends the address of a local account as name and fills no
// preferred_username. A sign in through the local provider keeps the address as the
// username of the identity.
func TestASignInThroughTheLocalProviderKeepsTheAddress(t *testing.T) {
	t.Parallel()

	fixture := profileUnderTest(t)
	nina := stringOf(t, fixture.me(t, auth.ProviderTelegram, ninaAccount)["id"])

	claims := fixture.provider.Claims(auth.ProviderLocal, nina)
	claims["name"] = ninaEmail
	delete(claims, "preferred_username")

	finished := fixture.signInWith(t, nina, claims)
	require.Equal(t, http.StatusFound, finished.Code)
	require.NotContains(t, finished.Header().Get("Location"), "error=")

	listed := byProvider(fixture.listIdentities(t, auth.ProviderLocal, nina))
	assert.Equal(t, ninaEmail, listed[auth.ProviderLocal]["username"])
}
