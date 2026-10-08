package handler_test

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authtest"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/dex/dextest"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	providers "github.com/abgeo/maroid/apps/hub/internal/provider"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	shellTarget   = "http://maroid.localhost"
	providerCloud = "cloud"
	flowLifetime  = 10 * time.Minute
	bindingCookie = auth.BindingCookieName
	sessionCookie = auth.SessionCookieName
)

type authFixture struct {
	router       chi.Router
	database     *sqlx.DB
	provider     *authtest.Provider
	identityRepo repository.IdentityRepository
	service      *auth.Service
	idp          *dextest.Memory
}

// authUnderTest builds the handler with every real dependency, so a test drives
// the routes the way a browser does.
func authUnderTest(t *testing.T) *authFixture {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	provider := authtest.StartProvider(t)

	cfg := &config.Config{}
	cfg.Auth.AllowedRedirects = []string{shellTarget}
	cfg.Auth.DeckURL = shellTarget
	cfg.Auth.FlowTTL = flowLifetime
	cfg.OIDC.Issuer = provider.URL
	cfg.OIDC.ClientID = authtest.ClientID
	cfg.OIDC.ClientSecret = "secret"
	cfg.OIDC.RedirectURI = "http://hub.maroid.localhost/auth/callback"

	oidcSvc, err := auth.NewOIDCService(cfg)
	require.NoError(t, err)

	identityRepo := repository.NewIdentity(instance.DB)
	invitationRepo := repository.NewInvitation(instance.DB)
	userRepo := repository.NewUser(instance.DB)
	service := auth.NewService(
		instance.DB, userRepo, identityRepo, invitationRepo,
		repository.NewWorkspace(instance.DB), repository.NewWorkspaceMember(instance.DB),
		repository.NewAllowedPlugin(instance.DB),
	)

	idp := authIDP()

	authHandler := handler.NewAuth(
		cfg,
		slog.New(slog.DiscardHandler),
		auth.NewTokenVerifier(oidcSvc),
		auth.NewOIDCFlow(oidcSvc, repository.NewAuthFlow(instance.DB), cfg.Auth.FlowTTL),
		userRepo,
		identityRepo,
		auth.NewResolver(identityRepo),
		invitationRepo,
		service,
		providers.NewManager(idp, provider.URL),
	)

	router := chi.NewRouter()
	router.Use(address.Middleware("https://hub.example.com"))
	authHandler.Register(router)

	return &authFixture{
		router:       router,
		database:     instance.DB,
		provider:     provider,
		identityRepo: identityRepo,
		service:      service,
		idp:          idp,
	}
}

// authIDP holds the two providers that the scenarios of EXTID name.
func authIDP() *dextest.Memory {
	return dextest.New(
		dextest.Connector(auth.ProviderTelegram, "oidc", "Telegram", `{"maroidPreset":"telegram"}`),
		dextest.Connector(providerCloud, "oidc", "ABGEO.cloud", `{"maroidPreset":"oidc"}`),
	)
}

// EXTID-SC-008: The attach lands on the record that the flow row names. The
// request that finishes the flow carries nothing that can move it.
// EXTID-FR-006 gives it.
func TestTheAttachLandsOnTheRecordThatStartedIt(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	ctx := t.Context()

	starter := addUserRecord(t, fixture.database, "Temuri")
	other := addUserRecord(t, fixture.database, "Nino")

	// The starter signs in with Telegram, so the middleware resolves to them.
	require.NoError(t, fixture.service.Attach(
		ctx, starter, auth.ProviderTelegram, "111", model.Profile{},
	))
	require.NoError(t, fixture.service.Attach(
		ctx, other, auth.ProviderTelegram, "222", model.Profile{},
	))

	link := httptest.NewRequestWithContext(ctx, http.MethodPost,
		"/auth/identities?provider="+providerCloud+"&redirect="+url.QueryEscape(shellTarget), nil)
	link.AddCookie(requestCookie(sessionCookie, fixture.provider.Sign(
		t, auth.ProviderTelegram, "111",
	)))

	started := httptest.NewRecorder()
	fixture.router.ServeHTTP(started, link)
	require.Equal(t, http.StatusAccepted, started.Code, "the starter names the provider")

	state := stateOf(t, started)
	binding := cookieOf(t, started, bindingCookie)

	claims := fixture.provider.Claims(providerCloud, "an-account")
	claims["nonce"] = nonceOf(t, fixture.database, state)
	fixture.provider.IssueCode("the-code", claims)

	// The callback is public. It carries a forged cookie that names the other
	// record, and a session cookie for the other record.
	callback := httptest.NewRequestWithContext(ctx, http.MethodGet,
		"/auth/callback?state="+state+"&code=the-code", nil)
	callback.AddCookie(requestCookie(bindingCookie, binding))
	callback.AddCookie(requestCookie("maroid_user_id", other))
	callback.AddCookie(requestCookie(sessionCookie, fixture.provider.Sign(
		t, auth.ProviderTelegram, "222",
	)))

	finished := httptest.NewRecorder()
	fixture.router.ServeHTTP(finished, callback)

	require.Equal(t, http.StatusFound, finished.Code)
	require.NotContains(t, finished.Header().Get("Location"), "error=")

	ofStarter, err := fixture.identityRepo.ListByUser(ctx, starter)
	require.NoError(t, err)
	require.Len(t, ofStarter, 2, "the attach lands on the record that started it")

	ofOther, err := fixture.identityRepo.ListByUser(ctx, other)
	require.NoError(t, err)
	require.Len(t, ofOther, 1, "no request moves the attach to another record")
}

// EXTID-SC-014: The redemption spends the invitation, writes the first identity,
// and starts the session.
// EXTID-SC-015: A second person who holds the same address reaches nothing.
func TestTheRedemptionBindsTheFirstIdentity(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	ctx := t.Context()

	issued, err := fixture.service.Invite(
		ctx, auth.InviteRequest{FirstName: "Nino"}, flowLifetime,
	)
	require.NoError(t, err)

	// The command prints an address of the deck. This stands in for the page of
	// the deck: it reads the token and hands it to the hub with its own landing
	// page as the target. EXTID-DD-015 gives the two steps.
	target := handOffToHub(t, issued.Token)

	started := httptest.NewRecorder()
	fixture.router.ServeHTTP(
		started,
		httptest.NewRequestWithContext(ctx, http.MethodPost, target, nil),
	)
	require.Equal(t, http.StatusAccepted, started.Code)

	state := stateOf(t, started)

	claims := fixture.provider.Claims(providerCloud, "the-account")
	claims["nonce"] = nonceOf(t, fixture.database, state)
	fixture.provider.IssueCode("redeem-code", claims)

	callback := httptest.NewRequestWithContext(ctx, http.MethodGet,
		"/auth/callback?state="+state+"&code=redeem-code", nil)
	callback.AddCookie(requestCookie(bindingCookie, cookieOf(t, started, bindingCookie)))

	finished := httptest.NewRecorder()
	fixture.router.ServeHTTP(finished, callback)

	require.Equal(t, http.StatusFound, finished.Code)
	require.NotContains(t, finished.Header().Get("Location"), "error=")
	require.NotEmpty(t, cookieOf(t, finished, sessionCookie), "the redemption signs them in")

	identities, err := fixture.identityRepo.ListByUser(ctx, issued.UserID)
	require.NoError(t, err)
	require.Len(t, identities, 1)

	// A spent invitation answers a problem, because the deck reaches this route
	// with fetch and reads no redirect. The deck then names the reason itself.
	replay := httptest.NewRecorder()
	fixture.router.ServeHTTP(
		replay,
		httptest.NewRequestWithContext(ctx, http.MethodPost, target, nil),
	)
	require.Equal(t, http.StatusNotFound, replay.Code)

	var failure problem.Problem

	require.NoError(t, json.Unmarshal(replay.Body.Bytes(), &failure))
	require.Equal(t, problem.TypeNotFound, failure.Type, "the grant is spent")
}

// EXTID-SC-011: The list names every provider that Dex holds. It
// marks the one that the record attached, with the handle that provider gave, and
// marks the other as not attached.
// EXTID-FR-009: A provider that nobody attached appears, because the person picks
// it from this list.
func TestTheListNamesEveryProvider(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	ctx := t.Context()

	userID := addUserRecord(t, fixture.database, "Temuri")
	require.NoError(t, fixture.service.Attach(
		ctx,
		userID,
		auth.ProviderTelegram,
		"111",
		model.Profile{Username: "abgeo", DisplayName: "Temuri", PictureURL: "https://a/b.jpg"},
	))

	body := byProvider(fixture.listIdentities(t, auth.ProviderTelegram, "111"))
	require.Len(t, body, 2, "Dex holds two providers")

	telegram := body[auth.ProviderTelegram]
	require.Equal(t, "Telegram", telegram["name"], "the deck shows this text")
	require.Equal(t, true, telegram["attached"])
	require.Equal(t, "abgeo", telegram["username"])
	require.Equal(t, "Temuri", telegram["display_name"])
	require.Equal(t, "https://a/b.jpg", telegram["picture_url"])
	require.NotEmpty(t, telegram["attached_at"])

	// APIFMT-SC-009: encoding/json writes a time.Time in the zone that it
	// carries, and the database answers the zone of the session.
	attachedAt, ok := telegram["attached_at"].(string)
	require.True(t, ok)
	require.True(t, strings.HasSuffix(attachedAt, "Z"), attachedAt)

	cloud := body[providerCloud]
	require.Equal(t, false, cloud["attached"], "a provider that nobody attached appears")
	require.Nil(t, cloud["username"])
	require.Nil(t, cloud["picture_url"])
}

// byProvider keys the list of identities by the provider of each item.
func byProvider(listed []map[string]any) map[string]map[string]any {
	keyed := make(map[string]map[string]any, len(listed))
	for _, item := range listed {
		if id, ok := item["provider"].(string); ok {
			keyed[id] = item
		}
	}

	return keyed
}

// EXTID-FR-009: The list belongs to the person that asks for it, and to no other.
func TestTheListHoldsNothingOfAnotherRecord(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	ctx := t.Context()

	mine := addUserRecord(t, fixture.database, "Temuri")
	theirs := addUserRecord(t, fixture.database, "Nino")

	require.NoError(t, fixture.service.Attach(
		ctx, mine, auth.ProviderTelegram, "111", model.Profile{},
	))
	require.NoError(t, fixture.service.Attach(
		ctx, theirs, providerCloud, "abc", model.Profile{Username: "not-mine"},
	))

	body := byProvider(fixture.listIdentities(t, auth.ProviderTelegram, "111"))

	require.Equal(t, true, body[auth.ProviderTelegram]["attached"], "my own provider")
	require.Equal(t, false, body[providerCloud]["attached"],
		"the account of another record is not mine")
	require.Nil(t, body[providerCloud]["username"])
}

// A request that carries no session reaches no list.
func TestTheListNeedsASession(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)

	recorder := httptest.NewRecorder()
	fixture.router.ServeHTTP(recorder, httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/auth/identities", nil,
	))

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

// APIFMT-SC-010: A member that holds no value is absent, and no answer carries a
// JSON null. A record whose owner set neither name answers neither member.
func TestMeOmitsANameThatTheRecordDoesNotHold(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)

	var userID string
	require.NoError(t, fixture.database.Get(
		&userID,
		`INSERT INTO public.users (first_name, last_name) VALUES (NULL, NULL) RETURNING id;`,
	))
	require.NoError(t, fixture.service.Attach(
		t.Context(), userID, auth.ProviderTelegram, "222", model.Profile{},
	))

	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/auth/sessions/self", nil,
	)
	request.AddCookie(requestCookie(
		sessionCookie, fixture.provider.Sign(t, auth.ProviderTelegram, "222"),
	))

	recorder := httptest.NewRecorder()
	fixture.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	require.NotContains(t, recorder.Body.String(), "null", recorder.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

	require.NotContains(t, body, "first_name")
	require.NotContains(t, body, "last_name")
	require.Contains(t, body, "provider", "a member that holds a value stays")
}

// GET /auth/sessions/self reports the two names of the record and the provider that
// authenticated the session, so the deck can warn before a detach that would
// sign the person out of their own request.
func TestMeReportsTheNameAndTheProvider(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	ctx := t.Context()

	var userID string
	require.NoError(t, fixture.database.Get(
		&userID,
		`INSERT INTO public.users (first_name, last_name) VALUES ($1, $2) RETURNING id;`,
		"Temuri", "Takalandze",
	))
	require.NoError(t, fixture.service.Attach(
		ctx, userID, auth.ProviderTelegram, "111", model.Profile{},
	))

	body := fixture.me(t, auth.ProviderTelegram, "111")

	require.Equal(t, "Temuri", body["first_name"])
	require.Equal(t, "Takalandze", body["last_name"])
	require.Equal(t, auth.ProviderTelegram, body["provider"])
}

// A record that holds two identities signs in with either one, and each
// session reports the provider that it actually used, not the other.
func TestMeNamesTheProviderThatSignedIn(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	ctx := t.Context()

	userID := addUserRecord(t, fixture.database, "Temuri")
	require.NoError(t, fixture.service.Attach(
		ctx, userID, auth.ProviderTelegram, "111", model.Profile{},
	))
	require.NoError(t, fixture.service.Attach(
		ctx, userID, providerCloud, "abc", model.Profile{},
	))

	require.Equal(
		t, auth.ProviderTelegram, fixture.me(t, auth.ProviderTelegram, "111")["provider"],
	)
	require.Equal(t, providerCloud, fixture.me(t, providerCloud, "abc")["provider"])
}

// APIFMT-SC-001, APIFMT-FR-009: A provider that gives no picture leaves the
// member out, as GET /auth/identities leaves out picture_url. One condition
// answers one way.
func TestMeOmitsAPictureThatTheProviderDidNotGive(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)

	userID := addUserRecord(t, fixture.database, "Temuri")
	require.NoError(t, fixture.service.Attach(
		t.Context(), userID, auth.ProviderTelegram, "111", model.Profile{},
	))

	claims := fixture.provider.Claims(auth.ProviderTelegram, "111")
	delete(claims, "picture")

	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/auth/sessions/self", nil,
	)
	request.AddCookie(requestCookie(sessionCookie, fixture.provider.SignClaims(t, claims)))

	recorder := httptest.NewRecorder()
	fixture.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var body map[string]any

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.NotContains(t, body, "picture", "no picture is no member, never an empty string")
	assert.Equal(t, auth.ProviderTelegram, body["provider"])
	assert.Equal(t, "https://example.com/a.jpg",
		fixture.me(t, auth.ProviderTelegram, "111")["picture"], "a picture that exists stays")
}

// me reads GET /auth/sessions/self as the person that the external account names.
func (f *authFixture) me(t *testing.T, connector string, accountID string) map[string]any {
	t.Helper()

	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/auth/sessions/self", nil,
	)
	request.AddCookie(requestCookie(sessionCookie, f.provider.Sign(t, connector, accountID)))

	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var body map[string]any

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

	return body
}

// listIdentities reads the route as the person that the external account names.
func (f *authFixture) listIdentities(
	t *testing.T,
	connector string,
	accountID string,
) []map[string]any {
	t.Helper()

	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/auth/identities", nil,
	)
	request.AddCookie(requestCookie(sessionCookie, f.provider.Sign(t, connector, accountID)))

	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	// The collection answers a page, and the items live under one member.
	// APIFMT-FR-002.
	var page struct {
		Items []map[string]any `json:"items"`
	}

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &page))

	body := page.Items

	return body
}

// signIn drives a whole sign in and returns the recorder of the callback.
func (f *authFixture) signIn(
	t *testing.T,
	connector string,
	accountID string,
) *httptest.ResponseRecorder {
	t.Helper()

	started := httptest.NewRecorder()
	f.router.ServeHTTP(started, httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/auth/sessions?redirect="+url.QueryEscape(shellTarget),
		nil,
	))
	require.Equal(t, http.StatusAccepted, started.Code)

	state := stateOf(t, started)

	claims := f.provider.Claims(connector, accountID)
	claims["nonce"] = nonceOf(t, f.database, state)
	f.provider.IssueCode("sign-in-code-"+accountID, claims)

	callback := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/auth/callback?state="+state+"&code=sign-in-code-"+accountID,
		nil,
	)
	callback.AddCookie(requestCookie(bindingCookie, cookieOf(t, started, bindingCookie)))

	finished := httptest.NewRecorder()
	f.router.ServeHTTP(finished, callback)

	return finished
}

// EXTID-SC-002: A sign in whose external account holds no identity writes no row.
// EXTID-FR-003: No sign in creates a user record or an identity.
// EXTID-SC-004: The refusal carries a reason of its own, so the deck asks the
// person to seek an invitation instead of offering a second attempt.
func TestASignInOfAnUnknownAccountWritesNothing(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)

	before := counts(t, fixture.database)

	finished := fixture.signIn(t, providerCloud, "an-account-that-nobody-holds")

	require.Equal(t, http.StatusFound, finished.Code)
	require.Contains(t, finished.Header().Get("Location"), "error=no_identity")
	require.NotContains(t, finished.Header().Get("Location"), "auth_failed")
	require.Empty(t, cookieOf(t, finished, sessionCookie), "no session starts")
	require.Equal(t, before, counts(t, fixture.database), "no row appears")
}

// EXTID-SC-003: An identity that names a record which is not active reaches
// nothing, and the reason differs from an account that holds no identity.
func TestASignInOfABlockedRecordIsRefused(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	ctx := t.Context()

	userID := addUserRecord(t, fixture.database, "Temuri")
	require.NoError(t, fixture.service.Attach(
		ctx, userID, providerCloud, "an-account", model.Profile{},
	))

	_, err := fixture.database.Exec(
		`UPDATE public.users SET status = $1 WHERE id = $2;`, model.StatusBlocked, userID,
	)
	require.NoError(t, err)

	finished := fixture.signIn(t, providerCloud, "an-account")

	require.Equal(t, http.StatusFound, finished.Code)
	require.Contains(t, finished.Header().Get("Location"), "error=access_denied")
	require.Empty(t, cookieOf(t, finished, sessionCookie))
}

// EXTID-SC-017: A sign in changes neither name of the user record.
// EXTID-SC-018: The identity carries what the provider gave at that sign in, and
// the identity of another provider does not change.
func TestASignInWritesTheIdentityAndNotTheNames(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	ctx := t.Context()

	userID := addUserRecord(t, fixture.database, "Temuri")
	require.NoError(t, fixture.service.Attach(
		ctx, userID, providerCloud, "an-account", model.Profile{Username: "old_handle"},
	))
	require.NoError(t, fixture.service.Attach(
		ctx, userID, auth.ProviderTelegram, "111", model.Profile{Username: "telegram_handle"},
	))

	finished := fixture.signIn(t, providerCloud, "an-account")
	require.Equal(t, http.StatusFound, finished.Code)
	require.NotContains(t, finished.Header().Get("Location"), "error=")

	var person struct {
		FirstName *string `db:"first_name"`
		LastName  *string `db:"last_name"`
	}

	require.NoError(t, fixture.database.Get(&person,
		`SELECT first_name, last_name FROM public.users WHERE id = $1;`, userID))
	require.Equal(t, "Temuri", *person.FirstName, "the sign in writes no name of the record")
	require.Nil(t, person.LastName)

	identities, err := fixture.identityRepo.ListByUser(ctx, userID)
	require.NoError(t, err)

	byProvider := map[string]string{}
	for _, identity := range identities {
		byProvider[identity.Provider] = *identity.Username
	}

	require.Equal(t, authtest.HandleOfA, byProvider[providerCloud], "the provider wrote its own")
	require.Equal(t, "telegram_handle", byProvider[auth.ProviderTelegram], "the other stays")
}

// counts reports how many rows the two tables of a sign in could gain.
func counts(t *testing.T, database *sqlx.DB) [2]int {
	t.Helper()

	var users, identities int

	require.NoError(t, database.Get(&users, `SELECT count(*) FROM public.users;`))
	require.NoError(t, database.Get(&identities, `SELECT count(*) FROM public.identities;`))

	return [2]int{users, identities}
}

// handOffToHub stands in for the invite page of the deck. It reads the token from
// the address that `maroid user invite` printed, then builds the request that the
// page makes to the hub, naming its own landing page as the target.
func handOffToHub(t *testing.T, token string) string {
	t.Helper()

	printed, err := auth.InviteAddress(shellTarget, token)
	require.NoError(t, err)

	parsed, err := url.Parse(printed)
	require.NoError(t, err)
	require.Equal(t, "/invite", parsed.Path, "the address belongs to the deck")

	landing := shellTarget + "/auth/callback"

	return "/auth/invitation-redemptions?token=" + url.QueryEscape(parsed.Query().Get("token")) +
		"&redirect=" + url.QueryEscape(landing)
}

// stateOf reads the state out of the address that a start answers. The three
// starts answer 202 with the address in the body, because a POST cannot redirect
// a browser that reached it with fetch. API-003.
func stateOf(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var handoff struct {
		AuthorizationURL string `json:"authorization_url"`
	}

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &handoff), recorder.Body.String())

	parsed, err := url.Parse(handoff.AuthorizationURL)
	require.NoError(t, err)

	state := parsed.Query().Get("state")
	require.NotEmpty(t, state)

	return state
}

func nonceOf(t *testing.T, database *sqlx.DB, state string) string {
	t.Helper()

	var nonce string

	require.NoError(t, database.Get(
		&nonce, `SELECT nonce FROM public.auth_flows WHERE state = $1;`, state,
	))

	return nonce
}

// requestCookie names a cookie that a browser sends back. gosec reads a literal
// http.Cookie as one that the server sets, and these travel the other way.
func requestCookie(name string, value string) *http.Cookie {
	//nolint:gosec // G124: a request carries a name and a value, and no attribute.
	return &http.Cookie{Name: name, Value: value}
}

func cookieOf(t *testing.T, recorder *httptest.ResponseRecorder, name string) string {
	t.Helper()

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == name && cookie.Value != "" {
			return cookie.Value
		}
	}

	return ""
}

func addUserRecord(t *testing.T, database *sqlx.DB, firstName string) string {
	t.Helper()

	var id string

	require.NoError(t, database.Get(
		&id, `INSERT INTO public.users (first_name) VALUES ($1) RETURNING id;`, firstName,
	))

	return id
}

// WSPACE-SC-021: The person of the session carries the identifier of the user
// record, so the deck names it to leave a workspace.
func TestTheCurrentUserCarriesTheRecordID(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)

	record := addUserRecord(t, fixture.database, "Gio")
	require.NoError(t, fixture.service.Attach(
		t.Context(), record, auth.ProviderTelegram, "303", model.Profile{},
	))

	assert.Equal(t, record, fixture.me(t, auth.ProviderTelegram, "303")["id"])
}

// PLUGACC-FR-001: The current user answers the mark of an administrator, so the deck
// shows the administration to an administrator alone.
func TestTheCurrentUserCarriesTheMarkOfAnAdministrator(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)

	record := addUserRecord(t, fixture.database, "Zura")
	require.NoError(t, fixture.service.Attach(
		t.Context(), record, auth.ProviderTelegram, "404", model.Profile{},
	))

	assert.Equal(t, false, fixture.me(t, auth.ProviderTelegram, "404")["is_administrator"])

	_, err := fixture.database.ExecContext(t.Context(),
		`UPDATE public.users SET is_administrator = true WHERE id = $1;`, record)
	require.NoError(t, err)

	assert.Equal(t, true, fixture.me(t, auth.ProviderTelegram, "404")["is_administrator"])
}

// IDPROV-SC-016: The list names every provider that Dex holds, the static one and
// the local one included, by the names that Dex holds.
func TestTheListReadsTheProvidersOfDex(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	ctx := t.Context()

	require.NoError(t, fixture.idp.CreateConnector(ctx,
		dextest.Connector("mock", "mockCallback", "Mock", `{}`)))
	require.NoError(t, fixture.idp.CreateConnector(ctx,
		dextest.Connector(auth.ProviderLocal, "local", "Maroid", `{"maroidPreset":"local"}`)))
	require.NoError(t, fixture.idp.DeleteConnector(ctx, providerCloud))

	userID := addUserRecord(t, fixture.database, "Ana")
	require.NoError(t, fixture.service.Attach(
		ctx, userID, auth.ProviderTelegram, "111", model.Profile{},
	))

	body := byProvider(fixture.listIdentities(t, auth.ProviderTelegram, "111"))
	require.Len(t, body, 3)

	assert.Equal(t, "Mock", body["mock"]["name"])
	assert.Equal(t, "Maroid", body[auth.ProviderLocal]["name"])
	assert.Equal(t, true, body[auth.ProviderTelegram]["attached"])
	assert.Equal(t, false, body["mock"]["attached"])
}

// IDPROV-SC-021: An attach through the local provider answers request-invalid,
// names the parameter, and starts no flow.
func TestAnAttachThroughTheLocalProviderIsRefused(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	ctx := t.Context()

	userID := addUserRecord(t, fixture.database, "Nina")
	require.NoError(t, fixture.service.Attach(
		ctx, userID, auth.ProviderTelegram, "111", model.Profile{},
	))

	request := httptest.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"/auth/identities?provider="+auth.ProviderLocal+"&redirect="+url.QueryEscape(
			shellTarget,
		),
		nil,
	)
	request.AddCookie(
		requestCookie(sessionCookie, fixture.provider.Sign(t, auth.ProviderTelegram, "111")),
	)

	recorder := httptest.NewRecorder()
	fixture.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())

	var failure problem.Problem

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &failure))
	assert.Equal(t, problem.TypeRequestInvalid, failure.Type)
	assert.Contains(t, failure.Detail, "provider")

	var flows int

	require.NoError(t, fixture.database.Get(&flows, `SELECT count(*) FROM public.auth_flows;`))
	assert.Zero(t, flows)
}

// IDPROV-SC-029: A Dex that does not answer makes the list answer not-ready.
func TestTheListAnswersNotReadyWithoutDex(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)

	userID := addUserRecord(t, fixture.database, "Ana")
	require.NoError(t, fixture.service.Attach(
		t.Context(), userID, auth.ProviderTelegram, "111", model.Profile{},
	))

	fixture.idp.Fail(errs.ErrIDPUnavailable)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/identities", nil)
	request.AddCookie(
		requestCookie(sessionCookie, fixture.provider.Sign(t, auth.ProviderTelegram, "111")),
	)

	recorder := httptest.NewRecorder()
	fixture.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code, recorder.Body.String())
}
