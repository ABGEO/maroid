package handler_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hubdatabase "github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/notification"
	"github.com/abgeo/maroid/apps/hub/internal/notification/telegram"
	"github.com/abgeo/maroid/apps/hub/internal/notification/telegramtest"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	tokenOne      = "111111:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	tokenTwo      = "222222:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
	topicReminder = "reminder"
	topicReport   = "report"
	topicOfP      = "digest"
	kindTelegram  = "telegram"
	caseNormal    = "normal"
	botName       = "Bot"
	fieldToken    = "bot_token"
	fieldChat     = "chat_id"
	fieldThread   = "thread_id"
	secretMask    = "******"
	keyCipherTag  = "enc:"

	memberFields      = "fields"
	memberDestination = "destination"
	memberEnabled     = "enabled"
	memberKind        = "kind"
	memberPluginID    = "plugin_id"
	memberTopicID     = "topic_id"
	memberProviderID  = "provider_id"
	memberRoutes      = "routes"
	memberUnrouted    = "unrouted"
)

var errNotProtected = errors.New("the value is not protected by the key")

// keyCipher protects a value with a readable form that names its key, and records
// each key that a write used.
type keyCipher struct {
	mu   sync.Mutex
	keys []secret.Key
}

func (c *keyCipher) Encrypt(_ context.Context, key secret.Key, plaintext string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.keys = append(c.keys, key)

	return keyCipherTag + string(key) + ":" + plaintext, nil
}

func (c *keyCipher) Decrypt(_ context.Context, key secret.Key, ciphertext string) (string, error) {
	plaintext, found := strings.CutPrefix(ciphertext, keyCipherTag+string(key)+":")
	if !found {
		return "", fmt.Errorf("%w: %s", errNotProtected, key)
	}

	return plaintext, nil
}

// lockedBuffer keeps every log line of the notification routes and of the dispatcher.
type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buffer.Write(p) //nolint:wrapcheck // a bytes.Buffer never fails
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buffer.String()
}

// notificationParts holds what the routes of the notification providers reach: a
// stub of Telegram, a cipher, and the dispatcher that a plugin sends through.
type notificationParts struct {
	database   *sqlx.DB
	stub       *telegramtest.Server
	cipher     *keyCipher
	kinds      *notification.KindRegistry
	topics     *registry.NotificationTopicRegistry
	lines      *lockedBuffer
	logger     *slog.Logger
	dispatcher *notification.Dispatcher
}

func newNotificationParts(t *testing.T, database *sqlx.DB) *notificationParts {
	t.Helper()

	stub := telegramtest.NewServer(t)
	kinds := notification.NewKindRegistry()
	require.NoError(t, kinds.Register(telegram.New(telego.WithAPIServer(stub.URL()))))

	topics := registry.NewNotificationTopicRegistry()
	require.NoError(t, topics.Register(
		pluginapi.ParsePluginID(probePluginID),
		pluginapi.NotificationTopic{ID: topicReminder, Name: "Reminder"},
		pluginapi.NotificationTopic{ID: topicReport, Name: "Report"},
	))
	require.NoError(t, topics.Register(
		pluginapi.ParsePluginID(pluginP),
		pluginapi.NotificationTopic{ID: topicOfP, Name: "Digest"},
	))

	lines := &lockedBuffer{}
	logger := slog.New(slog.NewJSONHandler(lines, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cipher := &keyCipher{}

	return &notificationParts{
		database:   database,
		stub:       stub,
		cipher:     cipher,
		kinds:      kinds,
		topics:     topics,
		lines:      lines,
		logger:     logger,
		dispatcher: notification.NewDispatcher(logger, topics, database, kinds, cipher),
	}
}

func (p *notificationParts) service(
	enablements notification.EnablementSource,
) *notification.Manager {
	return notification.NewManager(p.database, p.kinds, p.topics, enablements, p.cipher)
}

// notify sends the reminder of the probe plugin in the workspace, as a job does.
func (f *workspaceFixture) notify(t *testing.T, workspaceID string) error {
	t.Helper()

	//nolint:wrapcheck // the test reads the error that the dispatcher answered.
	return f.notifications.dispatcher.Send(
		pluginapi.ContextWithActingWorkspace(t.Context(), workspaceID),
		pluginapi.ParsePluginID(probePluginID),
		topicReminder,
		pluginapi.Message{Title: "Plant care due", Body: "E001 Office"},
	)
}

// addProvider creates the Telegram provider "Family bot" with tokenOne in H as Ana, and
// answers its identifier.
func (f *workspaceFixture) addProvider(t *testing.T) string {
	t.Helper()

	response := f.call(
		t,
		f.ana,
		http.MethodPost,
		f.inH("/notification-providers"),
		map[string]any{
			memberKind:   kindTelegram,
			memberName:   "Family bot",
			memberFields: map[string]any{fieldToken: tokenOne},
		},
		"",
	)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	return stringOf(t, decode(t, response)[memberID])
}

// addRoute creates a route in H as Ana, and answers its identifier.
func (f *workspaceFixture) addRoute(
	t *testing.T,
	topic string,
	provider string,
	chat string,
) string {
	t.Helper()

	response := f.call(t, f.ana, http.MethodPost, f.inH("/notification-routes"), map[string]any{
		memberPluginID: probePluginID, memberTopicID: topic, memberProviderID: provider,
		memberDestination: map[string]any{fieldChat: chat},
	}, "")
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	return stringOf(t, decode(t, response)[memberID])
}

func (f *workspaceFixture) callsOf(method string) []telegramtest.Call {
	var found []telegramtest.Call

	for _, call := range f.notifications.stub.Calls() {
		if call.Method == method {
			found = append(found, call)
		}
	}

	return found
}

func (f *workspaceFixture) chatsSent() []string {
	sent := f.callsOf("sendMessage")
	chats := make([]string, 0, len(sent))

	for _, call := range sent {
		chats = append(chats, call.Params[fieldChat])
	}

	return chats
}

// providersOfH lists the providers of H as Ana.
func (f *workspaceFixture) providersOfH(t *testing.T) []map[string]any {
	t.Helper()

	return items(t, f.call(t, f.ana, http.MethodGet, f.inH("/notification-providers"), nil, ""))
}

// topicOf answers the item of the topic list of H that names the topic of the probe.
func (f *workspaceFixture) topicOf(t *testing.T, who person, topic string) map[string]any {
	t.Helper()

	response := f.call(t, who, http.MethodGet, f.inH("/notification-topics"), nil, "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	for _, item := range items(t, response) {
		if item[memberPluginID] == probePluginID && item[memberTopicID] == topic {
			return item
		}
	}

	return nil
}

func routesOf(t *testing.T, topic map[string]any) []any {
	t.Helper()

	routes, ok := topic[memberRoutes].([]any)
	require.True(t, ok)

	return routes
}

// NOTIF-SC-004: A viewer lists the providers, and a held secret reads as the mask.
func TestAViewerListsTheProvidersWithNoSecret(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	fixture.addProvider(t)

	response := fixture.call(
		t,
		fixture.gio,
		http.MethodGet,
		fixture.inH("/notification-providers"),
		nil,
		"",
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	listed := items(t, response)
	require.Len(t, listed, 1)
	assert.Equal(t, "Family bot", listed[0][memberName])
	assert.Equal(t, kindTelegram, listed[0][memberKind])
	assert.Equal(t, true, listed[0][memberEnabled])
	assert.Equal(t, map[string]any{fieldToken: secretMask}, listed[0][memberFields])
	assert.NotContains(t, response.Body.String(), tokenOne)
	assert.NotContains(t, response.Body.String(), keyCipherTag)
}

// NOTIF-SC-005: The name holds 1 to 64 characters after the trim.
func TestTheNameOfAProviderHoldsSixtyFourCharacters(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		name   string
		status int
	}{
		caseNormal: {name: "Family bot", status: http.StatusCreated},
		"limit":    {name: " " + strings.Repeat("a", 64) + " ", status: http.StatusCreated},
		"over":     {name: strings.Repeat("a", 65), status: http.StatusUnprocessableEntity},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fixture := workspaceUnderTest(t)

			response := fixture.call(
				t,
				fixture.ana,
				http.MethodPost,
				fixture.inH("/notification-providers"),
				map[string]any{
					memberKind:   kindTelegram,
					memberName:   tc.name,
					memberFields: map[string]any{fieldToken: tokenOne},
				},
				"",
			)
			require.Equal(t, tc.status, response.Code, response.Body.String())

			listed := fixture.providersOfH(t)

			if tc.status == http.StatusCreated {
				require.Len(t, listed, 1)
				assert.Equal(t, strings.TrimSpace(tc.name), listed[0][memberName])

				return
			}

			assert.Contains(t, response.Body.String(), `"/name"`)
			assert.Empty(t, listed)
		})
	}
}

// NOTIF-SC-006: A change that sends the mask keeps the stored token, and checks no token.
func TestAChangeWithTheMaskKeepsTheToken(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)
	fixture.addRoute(t, topicReminder, provider, "-100")

	response := fixture.call(
		t,
		fixture.ana,
		http.MethodPatch,
		fixture.inH("/notification-providers/"+provider),
		map[string]any{
			memberName:   "Home bot",
			memberFields: map[string]any{fieldToken: secretMask},
		},
		"",
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, "Home bot", decode(t, response)[memberName])
	assert.Len(t, fixture.callsOf("getMe"), 1, "the change runs no second check")

	require.NoError(t, fixture.notify(t, fixture.h))

	sent := fixture.callsOf("sendMessage")
	require.Len(t, sent, 1)
	assert.Equal(t, tokenOne, sent[0].Token)
}

// NOTIF-SC-007: A removal of a provider removes its routes. Another workspace reads it as absent.
func TestARemovedProviderTakesItsRoutes(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)
	fixture.addRoute(t, topicReminder, provider, "-100")
	fixture.addRoute(t, topicReport, provider, "-100")

	workspaceB := fixture.create(t, fixture.nino, "B")

	response := fixture.call(
		t,
		fixture.nino,
		http.MethodDelete,
		"/workspaces/"+workspaceB+"/notification-providers/"+provider,
		nil,
		"",
	)
	assert.Equal(t, http.StatusNotFound, response.Code, response.Body.String())

	response = fixture.call(
		t,
		fixture.ana,
		http.MethodDelete,
		fixture.inH("/notification-providers/"+provider),
		nil,
		"",
	)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	for _, topic := range []string{topicReminder, topicReport} {
		item := fixture.topicOf(t, fixture.ana, topic)
		assert.Empty(t, routesOf(t, item))
		assert.Equal(t, true, item[memberUnrouted])
	}
}

// NOTIF-SC-008: A disabled provider delivers nothing and keeps its routes.
func TestADisabledProviderKeepsItsRoutes(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)
	fixture.addRoute(t, topicReminder, provider, "-100")

	response := fixture.call(
		t,
		fixture.ana,
		http.MethodPatch,
		fixture.inH("/notification-providers/"+provider),
		map[string]any{memberEnabled: false},
		"",
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	require.NoError(t, fixture.notify(t, fixture.h))
	assert.Empty(t, fixture.chatsSent())
	assert.Len(t, routesOf(t, fixture.topicOf(t, fixture.ana, topicReminder)), 1)

	response = fixture.call(
		t,
		fixture.ana,
		http.MethodPatch,
		fixture.inH("/notification-providers/"+provider),
		map[string]any{memberEnabled: true},
		"",
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	require.NoError(t, fixture.notify(t, fixture.h))
	assert.Equal(t, []string{"-100"}, fixture.chatsSent())
}

type refusedFieldCase struct {
	body    map[string]any
	pointer string
}

// refusedFieldCases holds the provider rows of the table of NOTIF-SC-009.
func refusedFieldCases() map[string]refusedFieldCase {
	return map[string]refusedFieldCase{
		"empty token": {
			body: map[string]any{
				memberKind: kindTelegram, memberName: botName, memberFields: map[string]any{},
			},
			pointer: "#/fields/bot_token",
		},
		"unknown field": {
			body: map[string]any{
				memberKind:   kindTelegram,
				memberName:   botName,
				memberFields: map[string]any{fieldToken: tokenOne, "proxy": "y"},
			},
			pointer: "#/fields/proxy",
		},
	}
}

// NOTIF-SC-009: A value that the schema of the kind refuses stores nothing, and the
// answer names the field.
func TestAFieldThatTheKindRefusesIsNamed(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	for name, tc := range refusedFieldCases() {
		response := fixture.call(
			t, fixture.ana, http.MethodPost, fixture.inH("/notification-providers"), tc.body, "",
		)
		require.Equal(t, http.StatusUnprocessableEntity, response.Code, name)
		assert.Contains(t, response.Body.String(), "/problems/hub/settings-invalid", name)
		assert.Contains(t, response.Body.String(), tc.pointer, name)
	}

	assert.Empty(t, fixture.providersOfH(t))

	provider := fixture.addProvider(t)

	response := fixture.call(
		t,
		fixture.ana,
		http.MethodPost,
		fixture.inH("/notification-routes"),
		map[string]any{
			memberPluginID: probePluginID, memberTopicID: topicReminder, memberProviderID: provider,
			memberDestination: map[string]any{fieldChat: "abc"},
		},
		"",
	)
	require.Equal(t, http.StatusUnprocessableEntity, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), "#/destination/chat_id")
	assert.Empty(t, routesOf(t, fixture.topicOf(t, fixture.ana, topicReminder)))
}

type verifyCase struct {
	prepare func(stub *telegramtest.Server)
	status  int
	problem string
}

// verifyCases holds the rows of the table of NOTIF-SC-010.
func verifyCases() map[string]verifyCase {
	return map[string]verifyCase{
		"accepted": {prepare: func(*telegramtest.Server) {}, status: http.StatusCreated},
		"refused": {
			prepare: func(stub *telegramtest.Server) { stub.Refuse("", "Unauthorized") },
			status:  http.StatusUnprocessableEntity, problem: "/problems/hub/delivery-refused",
		},
		"no answer": {
			prepare: func(stub *telegramtest.Server) { stub.Silence() },
			status:  http.StatusBadGateway, problem: "/problems/hub/messaging-unreachable",
		},
	}
}

// NOTIF-SC-010: Telegram decides whether the token stores.
func TestTelegramDecidesWhetherTheTokenStores(t *testing.T) {
	t.Parallel()

	for name, tc := range verifyCases() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fixture := workspaceUnderTest(t)
			tc.prepare(fixture.notifications.stub)

			response := fixture.call(
				t,
				fixture.ana,
				http.MethodPost,
				fixture.inH("/notification-providers"),
				map[string]any{
					memberKind:   kindTelegram,
					memberName:   botName,
					memberFields: map[string]any{fieldToken: tokenOne},
				},
				"",
			)
			require.Equal(t, tc.status, response.Code, response.Body.String())

			listed := items(
				t,
				fixture.call(
					t,
					fixture.ana,
					http.MethodGet,
					fixture.inH("/notification-providers"),
					nil,
					"",
				),
			)

			if tc.problem == "" {
				assert.Len(t, listed, 1)

				return
			}

			assert.Contains(t, response.Body.String(), tc.problem)
			assert.Empty(t, listed)

			if name == "refused" {
				assert.Equal(t, "Unauthorized", decode(t, response)["detail"])
			}
		})
	}
}

// NOTIF-SC-011: The topic list names each route with its destination, and marks an
// unrouted topic.
func TestTheTopicListNamesEachRoute(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)

	response := fixture.call(
		t,
		fixture.ana,
		http.MethodPost,
		fixture.inH("/notification-routes"),
		map[string]any{
			memberPluginID: probePluginID, memberTopicID: topicReminder, memberProviderID: provider,
			memberDestination: map[string]any{fieldChat: "-100", fieldThread: "2749"},
		},
		"",
	)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	reminder := fixture.topicOf(t, fixture.gio, topicReminder)
	require.NotNil(t, reminder)
	assert.Equal(t, false, reminder[memberUnrouted])
	assert.Equal(t, "Reminder", reminder[memberName])

	routes := routesOf(t, reminder)
	require.Len(t, routes, 1)

	route, ok := routes[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, provider, route[memberProviderID])
	assert.Equal(
		t,
		map[string]any{fieldChat: "-100", fieldThread: "2749"},
		route[memberDestination],
	)

	report := fixture.topicOf(t, fixture.gio, topicReport)
	require.NotNil(t, report)
	assert.Empty(t, routesOf(t, report))
	assert.Equal(t, true, report[memberUnrouted])
}

type routeCase struct {
	name    string
	body    map[string]any
	status  int
	pointer string
}

// routeCases holds the rows of the table of NOTIF-SC-012, in the order of the table.
func routeCases(provider string, providerOfB string) []routeCase {
	body := func(pluginID, topicID, providerID, chat string) map[string]any {
		return map[string]any{
			memberPluginID: pluginID, memberTopicID: topicID, memberProviderID: providerID,
			memberDestination: map[string]any{fieldChat: chat},
		}
	}

	return []routeCase{
		{
			name:   caseNormal,
			body:   body(probePluginID, topicReminder, provider, "-100"),
			status: http.StatusCreated,
		},
		{
			name:   "second route",
			body:   body(probePluginID, topicReminder, provider, "-200"),
			status: http.StatusCreated,
		},
		{
			name: "plugin not enabled", body: body(pluginP, topicOfP, provider, "-100"),
			status: http.StatusUnprocessableEntity, pointer: `"/plugin_id"`,
		},
		{
			name: "provider of B", body: body(probePluginID, topicReminder, providerOfB, "-300"),
			status: http.StatusUnprocessableEntity, pointer: `"/provider_id"`,
		},
	}
}

// NOTIF-SC-012: A route names an enabled plugin and a provider of the workspace.
func TestARouteNamesWhatTheWorkspaceHolds(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)

	workspaceB := fixture.create(t, fixture.nino, "B")
	response := fixture.call(
		t,
		fixture.nino,
		http.MethodPost,
		"/workspaces/"+workspaceB+"/notification-providers",
		map[string]any{
			memberKind:   kindTelegram,
			memberName:   "Their bot",
			memberFields: map[string]any{fieldToken: tokenTwo},
		},
		"",
	)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	providerOfB := stringOf(t, decode(t, response)[memberID])

	for _, tc := range routeCases(provider, providerOfB) {
		name := tc.name

		response := fixture.call(
			t,
			fixture.ana,
			http.MethodPost,
			fixture.inH("/notification-routes"),
			tc.body,
			"",
		)
		require.Equal(t, tc.status, response.Code, name+": "+response.Body.String())

		if tc.pointer != "" {
			assert.Contains(t, response.Body.String(), tc.pointer, name)
		}
	}

	assert.Len(t, routesOf(t, fixture.topicOf(t, fixture.ana, topicReminder)), 2)
}

// NOTIF-SC-013: A route that repeats another, whatever the order of its members, conflicts.
func TestARepeatedRouteConflicts(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)

	body := func(destination map[string]any) map[string]any {
		return map[string]any{
			memberPluginID: probePluginID, memberTopicID: topicReminder, memberProviderID: provider,
			memberDestination: destination,
		}
	}

	response := fixture.call(t, fixture.ana, http.MethodPost, fixture.inH("/notification-routes"),
		body(map[string]any{fieldChat: "-100", fieldThread: "2749"}), "")
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	response = fixture.call(t, fixture.ana, http.MethodPost, fixture.inH("/notification-routes"),
		body(map[string]any{fieldThread: "2749", fieldChat: "-100"}), "")
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), "/problems/hub/notification-route-exists")
	assert.Len(t, routesOf(t, fixture.topicOf(t, fixture.ana, topicReminder)), 1)
}

// NOTIF-SC-014: A changed destination receives the next send, and a removed route
// leaves the topic unrouted.
func TestAChangedRouteAndARemovedRoute(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)
	route := fixture.addRoute(t, topicReminder, provider, "-100")

	response := fixture.call(
		t,
		fixture.ana,
		http.MethodPatch,
		fixture.inH("/notification-routes/"+route),
		map[string]any{memberDestination: map[string]any{fieldChat: "-200"}},
		"",
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	require.NoError(t, fixture.notify(t, fixture.h))
	assert.Equal(t, []string{"-200"}, fixture.chatsSent())

	response = fixture.call(
		t,
		fixture.ana,
		http.MethodDelete,
		fixture.inH("/notification-routes/"+route),
		nil,
		"",
	)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	assert.Equal(t, true, fixture.topicOf(t, fixture.ana, topicReminder)[memberUnrouted])
}

// NOTIF-SC-015: A disabled route receives nothing, and the other route still does.
func TestADisabledRouteReceivesNothing(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)
	first := fixture.addRoute(t, topicReminder, provider, "-100")
	fixture.addRoute(t, topicReminder, provider, "-200")

	response := fixture.call(
		t,
		fixture.ana,
		http.MethodPatch,
		fixture.inH("/notification-routes/"+first),
		map[string]any{memberEnabled: false},
		"",
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	require.NoError(t, fixture.notify(t, fixture.h))
	assert.Equal(t, []string{"-200"}, fixture.chatsSent())
}

// NOTIF-SC-016: A manager tests a route, even a disabled one, and a viewer cannot.
func TestAManagerTestsARoute(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)
	accepted := fixture.addRoute(t, topicReminder, provider, "-100")
	refused := fixture.addRoute(t, topicReminder, provider, "-404")
	fixture.notifications.stub.Refuse("-404", "Bad Request: chat not found")

	response := fixture.call(
		t,
		fixture.ana,
		http.MethodPatch,
		fixture.inH("/notification-routes/"+accepted),
		map[string]any{memberEnabled: false},
		"",
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	response = fixture.call(
		t,
		fixture.ana,
		http.MethodPost,
		fixture.inH("/notification-routes/"+accepted+"/test-messages"),
		nil,
		"",
	)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	response = fixture.call(
		t,
		fixture.ana,
		http.MethodPost,
		fixture.inH("/notification-routes/"+refused+"/test-messages"),
		nil,
		"",
	)
	require.Equal(t, http.StatusUnprocessableEntity, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), "/problems/hub/delivery-refused")
	assert.Equal(t, "Bad Request: chat not found", decode(t, response)["detail"])
	assert.Equal(t, []string{"-100", "-404"}, fixture.chatsSent())

	response = fixture.call(
		t,
		fixture.gio,
		http.MethodPost,
		fixture.inH("/notification-routes/"+accepted+"/test-messages"),
		nil,
		"",
	)
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), "/problems/http/permission-denied")
	assert.Len(t, fixture.chatsSent(), 2)
}

// NOTIF-SC-017: The routes of a disabled plugin stay, and serve again at the next enablement.
func TestTheRoutesOfADisabledPluginStay(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)
	fixture.addRoute(t, topicReminder, provider, "-100")

	_, err := fixture.database.ExecContext(
		t.Context(),
		`DELETE FROM public.workspace_plugins WHERE workspace_id = $1 AND plugin_id = $2;`,
		fixture.h,
		probePluginID,
	)
	require.NoError(t, err)

	assert.Nil(t, fixture.topicOf(t, fixture.ana, topicReminder))
	assert.Equal(t, 1, fixture.routeRows(t))

	fixture.enable(t, fixture.h, probePluginID)

	assert.Len(t, routesOf(t, fixture.topicOf(t, fixture.ana, topicReminder)), 1)
	require.NoError(t, fixture.notify(t, fixture.h))
	assert.Equal(t, []string{"-100"}, fixture.chatsSent())
}

// NOTIF-SC-018: A route of a topic that the plugin no longer declares stays hidden.
func TestARouteOfAnUndeclaredTopicStaysHidden(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)

	ctx := pluginapi.ContextWithActingWorkspace(t.Context(), fixture.h)
	require.NoError(t, hubdatabase.WithScopeTx(ctx, fixture.database, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO public.notification_routes
			(provider_id, plugin_id, topic_id, destination) VALUES ($1, $2, 'legacy', '{"chat_id": "-100"}');`,
			provider, probePluginID)

		return err //nolint:wrapcheck // the test reports the error as it is
	}))

	response := fixture.call(
		t,
		fixture.gio,
		http.MethodGet,
		fixture.inH("/notification-topics"),
		nil,
		"",
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.NotContains(t, response.Body.String(), "legacy")
	assert.Equal(t, 1, fixture.routeRows(t))
}

// NOTIF-SC-024: The row holds the ciphertext of the token, under the key of the workspace.
func TestTheTokenStoresProtected(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	fixture.addProvider(t)

	var stored string

	ctx := pluginapi.ContextWithActingWorkspace(t.Context(), fixture.h)
	require.NoError(t, hubdatabase.WithScopeTx(ctx, fixture.database, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &stored,
			`SELECT fields -> 'bot_token' ->> 'value' FROM public.notification_providers;`)
	}))

	assert.Equal(t, keyCipherTag+string(secret.WorkspaceKey(fixture.h))+":"+tokenOne, stored)
	assert.Contains(t, fixture.notifications.cipher.keys, secret.WorkspaceKey(fixture.h))
}

// NOTIF-SC-025: No log line holds the token, whatever the request or the failure.
func TestNoLogLineHoldsTheToken(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)

	response := fixture.call(
		t,
		fixture.ana,
		http.MethodPatch,
		fixture.inH("/notification-providers/"+provider),
		map[string]any{memberFields: map[string]any{fieldToken: tokenTwo}},
		"",
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	route := fixture.addRoute(t, topicReminder, provider, "-100")
	fixture.call(
		t,
		fixture.ana,
		http.MethodPost,
		fixture.inH("/notification-routes/"+route+"/test-messages"),
		nil,
		"",
	)

	fixture.notifications.stub.Refuse("-100", "Forbidden: bot was blocked by the user")
	require.Error(t, fixture.notify(t, fixture.h))

	lines := fixture.notifications.lines.String()
	require.NotEmpty(t, lines)
	assert.NotContains(t, lines, tokenOne)
	assert.NotContains(t, lines, tokenTwo)
}

// NOTIF-SC-026: A change reaches each send that starts 1 second after its response.
func TestAChangeReachesTheNextSend(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)
	route := fixture.addRoute(t, topicReminder, provider, "-100")

	const runs = 20

	for run := range runs {
		chat := fmt.Sprintf("-%d", 200+run)

		response := fixture.call(
			t,
			fixture.ana,
			http.MethodPatch,
			fixture.inH("/notification-routes/"+route),
			map[string]any{memberDestination: map[string]any{fieldChat: chat}},
			"",
		)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())

		time.Sleep(time.Second)

		require.NoError(t, fixture.notify(t, fixture.h))

		sent := fixture.chatsSent()
		require.Equal(t, chat, sent[len(sent)-1], "run %d", run)
	}
}

// routeRows counts the routes of H with SQL, past what the routes show.
func (f *workspaceFixture) routeRows(t *testing.T) int {
	t.Helper()

	var count int

	ctx := pluginapi.ContextWithActingWorkspace(t.Context(), f.h)
	require.NoError(t, hubdatabase.WithScopeTx(ctx, f.database, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count, `SELECT count(*) FROM public.notification_routes;`)
	}))

	return count
}

// NOTIF-SC-030: The routes read one by one and as a list, and the list leaves out a
// route of an undeclared topic.
func TestTheRoutesReadOneByOneAndAsAList(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	provider := fixture.addProvider(t)

	response := fixture.call(t, fixture.ana, http.MethodPost, fixture.inH("/notification-routes"),
		map[string]any{
			memberPluginID: probePluginID, memberTopicID: topicReminder, memberProviderID: provider,
			memberDestination: map[string]any{fieldChat: "-100"},
		}, "")
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	route := stringOf(t, decode(t, response)[memberID])
	location := response.Header().Get("Location")
	assert.Equal(t, "https://hub.example.com"+fixture.inH("/notification-routes/"+route), location)

	ctx := pluginapi.ContextWithActingWorkspace(t.Context(), fixture.h)
	require.NoError(t, hubdatabase.WithScopeTx(ctx, fixture.database, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO public.notification_routes
			(provider_id, plugin_id, topic_id, destination) VALUES ($1, $2, 'legacy', '{"chat_id": "-100"}');`,
			provider, probePluginID)

		return err //nolint:wrapcheck // the test reports the error as it is
	}))

	listed := items(
		t,
		fixture.call(t, fixture.gio, http.MethodGet, fixture.inH("/notification-routes"), nil, ""),
	)
	require.Len(t, listed, 1)
	assert.Equal(t, route, listed[0][memberID])

	response = fixture.call(
		t,
		fixture.gio,
		http.MethodGet,
		fixture.inH("/notification-routes/"+route),
		nil,
		"",
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, route, decode(t, response)[memberID])
	assert.NotEmpty(t, response.Header().Get("ETag"))
}
