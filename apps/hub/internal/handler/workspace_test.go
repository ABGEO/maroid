package handler_test

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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
	hubdatabase "github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/user"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/testdb"
)

// scopedNotes is a table that a workspace scopes, so a test can show that a change
// of a membership leaves the records of a workspace in place.
const scopedNotes = `
CREATE TABLE public.workspace_test_notes (
    id           UUID NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    body         TEXT NOT NULL
);

ALTER TABLE public.workspace_test_notes ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.workspace_test_notes FORCE ROW LEVEL SECURITY;

CREATE POLICY workspace_test_notes_isolation ON public.workspace_test_notes
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);
`

const (
	memberName   = "name"
	memberUserID = "user_id"
	memberID     = "id"
	memberRole   = "role"

	roleManager = "manager"
	roleEditor  = "editor"
	roleViewer  = "viewer"
)

// person is a user record and the Telegram account that signs it in.
type person struct {
	id      string
	account string
}

// workspaceFixture holds the users of the scenarios of WSPACE and workspace H,
// with Ana, Beka, and Gio as its members. Nino is a member of nothing.
type workspaceFixture struct {
	router   chi.Router
	database *sqlx.DB
	provider *authtest.Provider
	ana      person
	beka     person
	gio      person
	nino     person
	h        string
	authSvc  *auth.Service
	// probeRuns counts the requests that reached the read route of the probe plugin.
	probeRuns *atomic.Int32
	// probeWrites counts the requests that reached the write route of the probe plugin.
	probeWrites *atomic.Int32
}

func workspaceUnderTest(t *testing.T) *workspaceFixture {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	_, err = instance.DB.ExecContext(t.Context(), scopedNotes)
	require.NoError(t, err)

	provider := authtest.StartProvider(t)
	identityRepo := repository.NewIdentity(instance.DB)
	userRepo := repository.NewUser(instance.DB)
	authSvc := authServiceOf(instance.DB, identityRepo, userRepo)

	runs, writes := &atomic.Int32{}, &atomic.Int32{}
	fixture := &workspaceFixture{
		router: workspaceRouter(
			t,
			instance,
			provider,
			identityRepo,
			usersOf(t, instance.DB, userRepo, authSvc),
			userRepo,
			runs,
			writes,
		),
		database:    instance.DB,
		provider:    provider,
		probeRuns:   runs,
		probeWrites: writes,
	}

	signable := func(name string, account string) person {
		id := addUserRecord(t, instance.DB, name)
		require.NoError(
			t,
			authSvc.Attach(t.Context(), id, auth.ProviderTelegram, account, model.Profile{}),
		)

		return person{id: id, account: account}
	}

	fixture.ana = signable("Ana", "101")
	fixture.beka = signable("Beka", "102")
	fixture.gio = signable("Gio", "103")
	fixture.nino = signable("Nino", "104")
	fixture.authSvc = authSvc

	fixture.h = fixture.create(t, fixture.ana, "H")
	fixture.add(t, fixture.ana, fixture.h, fixture.beka, roleEditor)
	fixture.add(t, fixture.ana, fixture.h, fixture.gio, roleViewer)
	fixture.enable(t, fixture.h, probePluginID)

	return fixture
}

// enable writes the enablement of the plugin in the workspace, as the owner of the
// tables.
func (f *workspaceFixture) enable(t *testing.T, workspaceID string, pluginID string) {
	t.Helper()

	_, err := f.database.ExecContext(t.Context(),
		`INSERT INTO public.workspace_plugins (workspace_id, plugin_id) VALUES ($1, $2);`,
		workspaceID, pluginID)
	require.NoError(t, err)
}

// administrator signs in Zura, an administrator who is a member of nothing. A test of
// the routes of an administrator calls it, so the other scenarios keep their world.
func (f *workspaceFixture) administrator(t *testing.T) person {
	t.Helper()

	id := addUserRecord(t, f.database, "Zura")
	require.NoError(
		t,
		f.authSvc.Attach(t.Context(), id, auth.ProviderTelegram, "105", model.Profile{}),
	)

	_, err := f.database.ExecContext(t.Context(),
		`UPDATE public.users SET is_administrator = true WHERE id = $1;`, id)
	require.NoError(t, err)

	return person{id: id, account: "105"}
}

// authServiceOf builds the auth service over the database.
func authServiceOf(
	database *sqlx.DB,
	identityRepo repository.IdentityRepository,
	userRepo repository.UserRepository,
) *auth.Service {
	return auth.NewService(
		database,
		userRepo,
		identityRepo,
		repository.NewInvitation(database),
		repository.NewWorkspace(database),
		repository.NewWorkspaceMember(database),
	)
}

// workspaceRouter mounts the handler of the workspaces, the settings routes, and the
// routes of the probe plugin with every real dependency, the way the hub mounts them
// on one router.
func workspaceRouter(
	t *testing.T,
	instance *testdb.Instance,
	provider *authtest.Provider,
	identityRepo repository.IdentityRepository,
	users *user.Manager,
	userRepo repository.UserRepository,
	probeRuns *atomic.Int32,
	probeWrites *atomic.Int32,
) *chi.Mux {
	t.Helper()

	memberRepo := repository.NewWorkspaceMember(instance.DB)
	access, enablements := probeAccessOf(t, instance.DB, memberRepo)

	logger := slog.New(slog.DiscardHandler)
	verifier := workspaceVerifier(t, provider)
	resolver := auth.NewResolver(identityRepo)

	router := chi.NewRouter()
	router.Use(address.Middleware("https://hub.example.com"))
	router.Use(precondition.IfMatch)

	handler.NewPluginWrapper(
		logger, verifier, resolver, noIdempotency{}, access,
		pluginapi.ParsePluginID(probePluginID),
		probeRoutes(instance.DB, probeRuns, probeWrites),
	).Register(router)

	handler.NewPlugin(
		logger, verifier, resolver,
		loadedPlugins(t), registry.NewUIRegistry(), registry.NewCapabilityRegistry(),
		&stubSettings{}, noIdempotency{}, access, repository.NewAllowedPlugin(instance.DB),
	).Register(router)

	manager := workspace.NewManager(
		instance.DB,
		repository.NewWorkspace(instance.DB),
		memberRepo,
		userRepo,
	)

	handler.NewWorkspace(
		logger,
		verifier,
		resolver,
		noIdempotency{},
		memberRepo,
		manager,
		access.Authorizer,
		manager,
		enablements,
		repository.NewWorkspace(instance.DB),
	).Register(router)

	handler.NewUser(
		logger, verifier, resolver, noIdempotency{}, users, "http://maroid.localhost",
	).Register(router)

	return router
}

// usersOf builds the service of the user records over the database, with the plugins
// P and Q loaded.
func usersOf(
	t *testing.T,
	database *sqlx.DB,
	users repository.UserRepository,
	inviter user.Inviter,
) *user.Manager {
	t.Helper()

	plugins := registry.NewPluginRegistry()
	require.NoError(
		t,
		plugins.Register(newStubPlugin(pluginP, "1.0.0"), newStubPlugin(pluginQ, "1.0.0")),
	)

	return user.NewManager(
		database,
		user.Repositories{
			Users:   users,
			Allowed: repository.NewAllowedPlugin(database),
		},
		inviter,
		loadedPlugins(t),
		time.Hour,
	)
}

// probeAccessOf holds the checks of a route of the probe plugin over the database, and
// the service of the enablements that the handler of the workspaces takes.
func probeAccessOf(
	t *testing.T,
	database *sqlx.DB,
	members repository.WorkspaceMemberRepository,
) (handler.WorkspaceAccess, *workspace.Enablements) {
	t.Helper()

	enablements := workspace.NewEnablements(
		repository.NewWorkspacePlugin(
			database,
		),
		repository.NewAllowedPlugin(database),
		loadedPlugins(t),
	)

	return handler.WorkspaceAccess{
		Members:     members,
		Enablements: enablements,
		Authorizer:  probeAuthorizer(t),
	}, enablements
}

// loadedPlugins answers the plugins P, Q, and the probe, as the hub loaded them.
func loadedPlugins(t *testing.T) *registry.PluginRegistry {
	t.Helper()

	plugins := registry.NewPluginRegistry()
	require.NoError(t, plugins.Register(
		newStubPlugin(pluginP, "1.0.0"),
		newStubPlugin(pluginQ, "1.0.0"),
		newStubPlugin(probePluginID, "1.0.0"),
	))

	return plugins
}

// workspaceVerifier verifies the session cookies that the provider signs.
func workspaceVerifier(t *testing.T, provider *authtest.Provider) auth.TokenVerifier {
	t.Helper()

	cfg := &config.Config{}
	cfg.OIDC.Issuer = provider.URL
	cfg.OIDC.ClientID = authtest.ClientID
	cfg.OIDC.ClientSecret = "workspace-client-secret"

	oidcSvc, err := auth.NewOIDCService(cfg)
	require.NoError(t, err)

	return auth.NewTokenVerifier(oidcSvc)
}

// call sends one request as the person, with an optional body and If-Match.
func (f *workspaceFixture) call(
	t *testing.T,
	who person,
	method string,
	path string,
	body any,
	ifMatch string,
) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader

	if body != nil {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)

		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}

	request := httptest.NewRequestWithContext(t.Context(), method, path, reader)
	request.AddCookie(
		requestCookie(sessionCookie, f.provider.Sign(t, auth.ProviderTelegram, who.account)),
	)

	if method == http.MethodPatch {
		request.Header.Set("Content-Type", "application/merge-patch+json")
	} else if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	if ifMatch != "" {
		request.Header.Set(precondition.IfMatchHeader, ifMatch)
	}

	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, request)

	return recorder
}

func (f *workspaceFixture) create(t *testing.T, who person, name string) string {
	t.Helper()

	response := f.call(t, who, http.MethodPost, "/workspaces", map[string]any{memberName: name}, "")
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	return stringOf(t, decode(t, response)[memberID])
}

func (f *workspaceFixture) add(
	t *testing.T,
	who person,
	workspaceID string,
	member person,
	role string,
) {
	t.Helper()

	response := f.call(t, who, http.MethodPost, "/workspaces/"+workspaceID+"/members",
		map[string]any{memberUserID: member.id, memberRole: role}, "")
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
}

// memberIDs reads the members of the workspace as the person.
func (f *workspaceFixture) memberIDs(t *testing.T, who person, workspaceID string) []string {
	t.Helper()

	response := f.call(t, who, http.MethodGet, "/workspaces/"+workspaceID+"/members", nil, "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	listed := items(t, response)

	ids := make([]string, 0, len(listed))
	for _, item := range listed {
		ids = append(ids, stringOf(t, item[memberUserID]))
	}

	return ids
}

// writeNotes stores notes in the workspace, as a plugin would in its own schema.
func (f *workspaceFixture) writeNotes(t *testing.T, workspaceID string, bodies ...string) {
	t.Helper()

	ctx := pluginapi.ContextWithActingWorkspace(t.Context(), workspaceID)

	require.NoError(t, hubdatabase.WithScopeTx(ctx, f.database, func(tx *sqlx.Tx) error {
		for _, body := range bodies {
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO public.workspace_test_notes (body) VALUES ($1);`,
				body,
			); err != nil {
				return err //nolint:wrapcheck // the test reads the error as it is.
			}
		}

		return nil
	}))
}

func (f *workspaceFixture) countNotes(t *testing.T, workspaceID string) int {
	t.Helper()

	ctx := pluginapi.ContextWithActingWorkspace(t.Context(), workspaceID)

	var count int

	require.NoError(t, hubdatabase.WithScopeTx(ctx, f.database, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count, `SELECT count(*) FROM public.workspace_test_notes;`)
	}))

	return count
}

// stringOf reads a member of a decoded body that must be a string.
func stringOf(t *testing.T, value any) string {
	t.Helper()

	text, ok := value.(string)
	require.True(t, ok, "the member %v is not a string", value)

	return text
}

func decode(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))

	return body
}

func items(t *testing.T, response *httptest.ResponseRecorder) []map[string]any {
	t.Helper()

	var page struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &page))

	return page.Items
}

func requirePointer(t *testing.T, response *httptest.ResponseRecorder, pointer string) {
	t.Helper()

	require.Equal(t, http.StatusUnprocessableEntity, response.Code, response.Body.String())

	failures, ok := decode(t, response)["errors"].([]any)
	require.True(t, ok)
	require.Len(t, failures, 1)

	failure, ok := failures[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, pointer, failure["pointer"])
}

// WSPACE-SC-001: A person creates a workspace and becomes its one member. A name of
// 64 characters lands, and an empty name or one of 65 characters writes no row.
func TestAPersonCreatesAWorkspace(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	home := fixture.call(
		t,
		fixture.nino,
		http.MethodPost,
		"/workspaces",
		map[string]any{memberName: "Home"},
		"",
	)
	require.Equal(t, http.StatusCreated, home.Code, home.Body.String())

	created := decode(t, home)
	homeID := stringOf(t, created[memberID])

	assert.Equal(t, "Home", created[memberName])
	assert.Equal(t, "https://hub.example.com/workspaces/"+homeID, home.Header().Get("Location"))
	assert.NotEmpty(t, home.Header().Get(precondition.ETagHeader))
	assert.Equal(t, []string{fixture.nino.id}, fixture.memberIDs(t, fixture.nino, homeID))

	longest := fixture.call(t, fixture.nino, http.MethodPost, "/workspaces",
		map[string]any{memberName: strings.Repeat("ბ", 64)}, "")
	require.Equal(t, http.StatusCreated, longest.Code, longest.Body.String())

	for _, name := range []string{"", strings.Repeat("ბ", 65)} {
		requirePointer(t, fixture.call(t, fixture.nino, http.MethodPost, "/workspaces",
			map[string]any{memberName: name}, ""), "/name")
	}

	listed := fixture.call(t, fixture.nino, http.MethodGet, "/workspaces", nil, "")
	require.Equal(t, http.StatusOK, listed.Code)
	assert.Len(t, items(t, listed), 2)
}

// WSPACE-SC-002: A person lists the workspaces that they are a member of, and no
// other.
func TestAPersonListsTheirWorkspaces(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	garden := fixture.create(t, fixture.gio, "G")
	fixture.create(t, fixture.nino, "N")

	response := fixture.call(t, fixture.gio, http.MethodGet, "/workspaces", nil, "")
	require.Equal(t, http.StatusOK, response.Code)

	listed := items(t, response)
	require.Len(t, listed, 2)
	assert.Equal(t, garden, listed[0][memberID])
	assert.Equal(t, fixture.h, listed[1][memberID])
}

// WSPACE-SC-003: A rename lands, a name of 65 characters does not, and a rename
// with a validator from before the first one answers 412.
func TestAMemberRenamesTheWorkspace(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	path := "/workspaces/" + fixture.h

	before := fixture.call(t, fixture.ana, http.MethodGet, path, nil, "")
	require.Equal(t, http.StatusOK, before.Code)

	staleTag := before.Header().Get(precondition.ETagHeader)
	require.NotEmpty(t, staleTag)

	renamed := fixture.call(
		t,
		fixture.ana,
		http.MethodPatch,
		path,
		map[string]any{memberName: "Flat"},
		"",
	)
	require.Equal(t, http.StatusOK, renamed.Code, renamed.Body.String())
	assert.Equal(t, "Flat", decode(t, renamed)[memberName])

	requirePointer(t, fixture.call(t, fixture.ana, http.MethodPatch, path,
		map[string]any{memberName: strings.Repeat("ბ", 65)}, ""), "/name")

	stale := fixture.call(
		t,
		fixture.ana,
		http.MethodPatch,
		path,
		map[string]any{memberName: "Old"},
		staleTag,
	)
	assert.Equal(t, http.StatusPreconditionFailed, stale.Code, stale.Body.String())

	after := fixture.call(t, fixture.ana, http.MethodGet, path, nil, "")
	assert.Equal(t, "Flat", decode(t, after)[memberName])
}

// WSPACE-SC-004: The candidates are every active user record that is no member of
// the workspace.
func TestTheCandidatesAreTheActiveNonMembers(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	levan := addUserRecord(t, fixture.database, "Levan")
	_, err := fixture.database.ExecContext(t.Context(),
		`UPDATE public.users SET status = 'blocked' WHERE id = $1;`, levan)
	require.NoError(t, err)

	response := fixture.call(
		t,
		fixture.ana,
		http.MethodGet,
		"/workspaces/"+fixture.h+"/member-candidates",
		nil,
		"",
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	listed := items(t, response)
	require.Len(t, listed, 1)
	assert.Equal(t, fixture.nino.id, listed[0][memberUserID])
	assert.Equal(t, "Nino", listed[0]["first_name"])
}

// WSPACE-SC-005: A member adds a person one time. A second addition answers
// member-exists, and a blocked record answers a validation failure.
// WSPACE-INV-002 holds: the workspace keeps one row for the person.
func TestAMemberAddsAPersonOneTime(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	path := "/workspaces/" + fixture.h + "/members"

	levan := addUserRecord(t, fixture.database, "Levan")
	_, err := fixture.database.ExecContext(t.Context(),
		`UPDATE public.users SET status = 'blocked' WHERE id = $1;`, levan)
	require.NoError(t, err)

	added := fixture.call(
		t,
		fixture.ana,
		http.MethodPost,
		path,
		map[string]any{memberUserID: fixture.nino.id, memberRole: roleEditor},
		"",
	)
	require.Equal(t, http.StatusCreated, added.Code, added.Body.String())
	assert.Equal(
		t,
		"https://hub.example.com"+path+"/"+fixture.nino.id,
		added.Header().Get("Location"),
	)

	again := fixture.call(
		t,
		fixture.ana,
		http.MethodPost,
		path,
		map[string]any{memberUserID: fixture.nino.id, memberRole: roleEditor},
		"",
	)
	require.Equal(t, http.StatusConflict, again.Code)
	assert.Equal(t, "/problems/hub/member-exists", decode(t, again)["type"])

	requirePointer(
		t,
		fixture.call(
			t,
			fixture.ana,
			http.MethodPost,
			path,
			map[string]any{memberUserID: levan, memberRole: roleEditor},
			"",
		),
		"/user_id",
	)

	count := 0

	for _, id := range fixture.memberIDs(t, fixture.ana, fixture.h) {
		if id == fixture.nino.id {
			count++
		}
	}

	assert.Equal(t, 1, count)
}

// WSPACE-SC-006: A removal and a leave each take one membership out, and every
// record of the workspace stays.
func TestARemovalAndALeaveKeepTheRecords(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	fixture.writeNotes(t, fixture.h, "one", "two", "from Beka")

	removed := fixture.call(t, fixture.ana, http.MethodDelete,
		"/workspaces/"+fixture.h+"/members/"+fixture.beka.id, nil, "")
	require.Equal(t, http.StatusNoContent, removed.Code, removed.Body.String())

	left := fixture.call(t, fixture.gio, http.MethodDelete,
		"/workspaces/"+fixture.h+"/members/"+fixture.gio.id, nil, "")
	require.Equal(t, http.StatusNoContent, left.Code, left.Body.String())

	assert.Equal(t, []string{fixture.ana.id}, fixture.memberIDs(t, fixture.ana, fixture.h))
	assert.Equal(t, 3, fixture.countNotes(t, fixture.h))
}

// WSPACE-SC-007: A member reads every member of the workspace, and one membership
// with its validator.
func TestAMemberReadsTheMembers(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	assert.Equal(t,
		[]string{fixture.ana.id, fixture.beka.id, fixture.gio.id},
		fixture.memberIDs(t, fixture.gio, fixture.h),
	)

	one := fixture.call(t, fixture.gio, http.MethodGet,
		"/workspaces/"+fixture.h+"/members/"+fixture.ana.id, nil, "")
	require.Equal(t, http.StatusOK, one.Code, one.Body.String())
	assert.Equal(t, fixture.ana.id, decode(t, one)[memberUserID])
	assert.NotEmpty(t, one.Header().Get(precondition.ETagHeader))
}

// WSPACE-SC-008: A member leaves a workspace that keeps another member, and the
// records of the workspace stay for the member who remains.
func TestAMemberLeavesAndTheRecordsStay(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	garden := fixture.create(t, fixture.ana, "G")
	fixture.add(t, fixture.ana, garden, fixture.gio, roleViewer)
	fixture.writeNotes(t, garden, "from Ana", "from Gio")

	left := fixture.call(
		t,
		fixture.gio,
		http.MethodDelete,
		"/workspaces/"+garden+"/members/"+fixture.gio.id,
		nil,
		"",
	)
	require.Equal(t, http.StatusNoContent, left.Code, left.Body.String())

	assert.Equal(t, []string{fixture.ana.id}, fixture.memberIDs(t, fixture.ana, garden))
	assert.Equal(t, 2, fixture.countNotes(t, garden))

	listed := fixture.call(t, fixture.gio, http.MethodGet, "/workspaces", nil, "")
	for _, item := range items(t, listed) {
		assert.NotEqual(t, garden, item[memberID])
	}
}

// PERMS-SC-003: The person who creates a workspace is its manager.
func TestTheCreatorOfAWorkspaceIsItsManager(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	home := fixture.create(t, fixture.nino, "Home")

	var role string

	require.NoError(t, fixture.database.Get(&role,
		`SELECT role FROM public.workspace_members WHERE workspace_id = $1 AND user_id = $2;`,
		home, fixture.nino.id))
	assert.Equal(t, "manager", role)
}
