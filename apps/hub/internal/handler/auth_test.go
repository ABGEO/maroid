package handler_test

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authtest"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	shellTarget     = "http://maroid.localhost"
	providerCloud   = "cloud"
	flowLifetime    = 10 * time.Minute
	sessionLifetime = 168 * time.Hour
	bindingCookie   = "maroid_auth_binding"
	sessionCookie   = "maroid_token"
)

type authFixture struct {
	router       chi.Router
	database     *sqlx.DB
	provider     *authtest.Provider
	identityRepo repository.IdentityRepository
	service      *auth.Service
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
	cfg.Auth.FlowTTL = flowLifetime
	cfg.Auth.SessionTTL = sessionLifetime
	cfg.Auth.Providers = []config.Provider{
		{ID: auth.ProviderTelegram, Name: "Telegram"},
		{ID: providerCloud, Name: "ABGEO.cloud"},
	}
	cfg.OIDC.Issuer = provider.URL
	cfg.OIDC.ClientID = authtest.ClientID
	cfg.OIDC.ClientSecret = "secret"
	cfg.OIDC.RedirectURI = "http://hub.maroid.localhost/auth/callback"

	oidcSvc, err := auth.NewOIDCService(cfg)
	require.NoError(t, err)

	identityRepo := repository.NewIdentity(instance.DB)
	invitationRepo := repository.NewInvitation(instance.DB)
	userRepo := repository.NewUser(instance.DB)
	service := auth.NewService(instance.DB, userRepo, identityRepo, invitationRepo)

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
	)

	router := chi.NewRouter()
	authHandler.Register(router)

	return &authFixture{
		router:       router,
		database:     instance.DB,
		provider:     provider,
		identityRepo: identityRepo,
		service:      service,
	}
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

	link := httptest.NewRequestWithContext(ctx, http.MethodGet,
		"/auth/link?provider="+providerCloud+"&redirect="+url.QueryEscape(shellTarget), nil)
	link.Header.Set("Authorization", "Bearer "+fixture.provider.Sign(
		t, auth.ProviderTelegram, "111",
	))

	started := httptest.NewRecorder()
	fixture.router.ServeHTTP(started, link)
	require.Equal(t, http.StatusFound, started.Code, "the starter reaches the provider")

	state := stateOf(t, started)
	binding := cookieOf(t, started, bindingCookie)

	claims := fixture.provider.Claims(providerCloud, "an-account")
	claims["nonce"] = nonceOf(t, fixture.database, state)
	fixture.provider.IssueCode("the-code", claims)

	// The callback is public. It carries a forged cookie that names the other
	// record, and a token for the other record in the header.
	callback := httptest.NewRequestWithContext(ctx, http.MethodGet,
		"/auth/callback?state="+state+"&code=the-code", nil)
	callback.AddCookie(requestCookie(bindingCookie, binding))
	callback.AddCookie(requestCookie("maroid_user_id", other))
	callback.Header.Set("Authorization", "Bearer "+fixture.provider.Sign(
		t, auth.ProviderTelegram, "222",
	))

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
	address := handOffToHub(t, issued.Token)

	started := httptest.NewRecorder()
	fixture.router.ServeHTTP(
		started,
		httptest.NewRequestWithContext(ctx, http.MethodGet, address, nil),
	)
	require.Equal(t, http.StatusFound, started.Code)

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

	// A spent invitation reports at the target, so the deck renders it.
	replay := httptest.NewRecorder()
	fixture.router.ServeHTTP(
		replay,
		httptest.NewRequestWithContext(ctx, http.MethodGet, address, nil),
	)
	require.Equal(t, http.StatusFound, replay.Code)
	require.Contains(
		t,
		replay.Header().Get("Location"),
		"error=invitation_invalid",
		"the grant is spent, and the deck reads the reason",
	)
}

// EXTID-SC-011: The list names every provider that the configuration holds. It
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

	body := fixture.listIdentities(t, auth.ProviderTelegram, "111")
	require.Len(t, body, 2, "the configuration holds two providers")

	require.Equal(t, auth.ProviderTelegram, body[0]["provider"])
	require.Equal(t, "Telegram", body[0]["name"], "the deck shows this text")
	require.Equal(t, true, body[0]["attached"])
	require.Equal(t, "abgeo", body[0]["username"])
	require.Equal(t, "Temuri", body[0]["display_name"])
	require.Equal(t, "https://a/b.jpg", body[0]["picture_url"])
	require.NotEmpty(t, body[0]["attached_at"])

	require.Equal(t, providerCloud, body[1]["provider"])
	require.Equal(t, false, body[1]["attached"], "a provider that nobody attached appears")
	require.Nil(t, body[1]["username"])
	require.Nil(t, body[1]["picture_url"])
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

	body := fixture.listIdentities(t, auth.ProviderTelegram, "111")

	require.Equal(t, true, body[0]["attached"], "my own provider")
	require.Equal(t, false, body[1]["attached"], "the account of another record is not mine")
	require.Nil(t, body[1]["username"])
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

// /auth/me reports the two names of the record and the provider that
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

// me reads /auth/me as the person that the external account names.
func (f *authFixture) me(t *testing.T, connector string, accountID string) map[string]any {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+f.provider.Sign(t, connector, accountID))

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
	request.Header.Set("Authorization", "Bearer "+f.provider.Sign(t, connector, accountID))

	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var body []map[string]any

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

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
		http.MethodGet,
		"/auth?redirect="+url.QueryEscape(shellTarget),
		nil,
	))
	require.Equal(t, http.StatusFound, started.Code)

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

	return "/auth/invite?token=" + url.QueryEscape(parsed.Query().Get("token")) +
		"&redirect=" + url.QueryEscape(landing)
}

func stateOf(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	parsed, err := url.Parse(recorder.Header().Get("Location"))
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
