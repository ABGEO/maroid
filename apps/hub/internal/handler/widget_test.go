package handler_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/widget"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	thingsRead   = "things.read"
	thingsWrite  = "things.write"
	memberKey    = "key"
	memberState  = "state"
	memberStat   = "stat"
	memberValue  = "value"
	memberType   = "type"
	widgetOpen   = "open"
	widgetClosed = "closed"
	stateReady   = "ready"
	typeNotFound = "/problems/http/not-found"
)

var errParkingDown = errors.New("the parking service answered 503")

// widgetBench holds workspace H with the routes of the widgets mounted over the
// widgets that the test declares, and the log lines of the handler.
type widgetBench struct {
	*workspaceFixture

	// setup keeps the router of the workspaces, for a test that changes a membership.
	setup *workspaceFixture
	lines *lockedBuffer
}

// widgetAuthorizer answers the permissions of the hub, and things.read for a viewer
// and things.write for an editor in the plugins P and Q.
func widgetAuthorizer(t *testing.T) *authz.RoleAuthorizer {
	t.Helper()

	permissions, err := authz.NewPermissionRegistry()
	require.NoError(t, err)

	for _, id := range []string{pluginP, pluginQ} {
		plugin := pluginapi.ParsePluginID(id)
		require.NoError(t, permissions.Register(
			registry.PermissionEntry{
				Name: registry.PermissionName(plugin, thingsRead), Lowest: pluginapi.RoleViewer,
			},
			registry.PermissionEntry{
				Name: registry.PermissionName(plugin, thingsWrite), Lowest: pluginapi.RoleEditor,
			},
		))
	}

	return authz.NewRoleAuthorizer(permissions)
}

// staticWidget declares a stat widget of the plugin under the permission, whose
// Load answers the value.
func staticWidget(pluginID, key, permission, value string) registry.Widget {
	return loadingWidget(pluginID, key, permission,
		func(context.Context) (pluginapi.WidgetAnswer, error) {
			return pluginapi.StatAnswer(pluginapi.StatData{Value: value}), nil
		})
}

func loadingWidget(
	pluginID string,
	key string,
	permission string,
	load func(context.Context) (pluginapi.WidgetAnswer, error),
) registry.Widget {
	return registry.Widget{
		PluginID: pluginID,
		Meta: pluginapi.WidgetMeta{
			Key:   key,
			Title: "Things",
			Kind:  pluginapi.WidgetKindStat,
			Size:  pluginapi.WidgetSizeS,
		},
		Permission: registry.PermissionName(pluginapi.ParsePluginID(pluginID), permission),
		Load:       load,
	}
}

func widgetsUnderTest(
	t *testing.T,
	deadline time.Duration,
	widgets ...registry.Widget,
) *widgetBench {
	t.Helper()

	fixture := workspaceUnderTest(t)

	held := registry.NewWidgetRegistry()
	require.NoError(t, held.Register(widgets...))

	lines := &lockedBuffer{}
	logger := slog.New(slog.NewJSONHandler(lines, nil))
	enablements := workspace.NewEnablements(fixture.database, loadedPlugins(t))

	router := baseRouter()
	handler.NewWidget(
		logger,
		workspaceVerifier(t, fixture.provider),
		auth.NewResolver(fixture.database),
		handler.WorkspaceAccess{
			DB:          fixture.database,
			Enablements: enablements,
			Authorizer:  widgetAuthorizer(t),
		},
		enablements,
		held,
		widget.New(deadline),
	).Register(router)

	mounted := *fixture
	mounted.router = router

	return &widgetBench{workspaceFixture: &mounted, setup: fixture, lines: lines}
}

func (b *widgetBench) widgetsIn(t *testing.T, who person, workspaceID string) []string {
	t.Helper()

	response := b.call(t, who, http.MethodGet, "/workspaces/"+workspaceID+"/widgets", nil, "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	listed := items(t, response)

	keys := make([]string, 0, len(listed))
	for _, item := range listed {
		keys = append(keys, stringOf(t, item[memberKey]))
	}

	return keys
}

// read answers the data of one widget of P.
func (b *widgetBench) read(
	t *testing.T,
	who person,
	workspaceID string,
	key string,
) map[string]any {
	t.Helper()

	response := b.readRaw(t, who, workspaceID, pluginP, key)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	return decode(t, response)
}

func (b *widgetBench) readRaw(
	t *testing.T,
	who person,
	workspaceID string,
	pluginID string,
	key string,
) *httptest.ResponseRecorder {
	t.Helper()

	return b.call(t, who, http.MethodGet,
		"/workspaces/"+workspaceID+"/plugins/"+pluginID+"/widgets/"+key, nil, "")
}

func valueOf(t *testing.T, answer map[string]any) string {
	t.Helper()

	stat, ok := answer[memberStat].(map[string]any)
	require.True(t, ok, "the answer carries no stat: %v", answer)

	return stringOf(t, stat[memberValue])
}

// WIDGET-SC-005: The list holds the widgets of the plugins that the workspace
// enables, and a widget of another plugin reads as absent.
func TestTheListHoldsTheWidgetsOfTheEnabledPlugins(t *testing.T) {
	t.Parallel()

	bench := widgetsUnderTest(t, widget.Deadline,
		staticWidget(pluginP, "of-p", thingsRead, "p"),
		staticWidget(pluginQ, "of-q", thingsRead, "q"),
	)
	bench.enable(t, bench.h, pluginP)

	require.Equal(t, []string{"of-p"}, bench.widgetsIn(t, bench.ana, bench.h))

	listed := items(t, bench.call(t, bench.ana, http.MethodGet, bench.inH("/widgets"), nil, ""))
	assert.Equal(t, map[string]any{
		memberPluginID: pluginP,
		memberKey:      "of-p",
		"title":        "Things",
		"kind":         "stat",
		"size":         "s",
		"permission":   pluginP + ":" + thingsRead,
	}, listed[0])

	response := bench.readRaw(t, bench.ana, bench.h, pluginQ, "of-q")
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
	require.Equal(t, typeNotFound, decode(t, response)[memberType])

	response = bench.readRaw(t, bench.ana, bench.h, pluginP, "absent")
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
	require.Equal(t, typeNotFound, decode(t, response)[memberType])
}

// WIDGET-SC-030: The widgets of one plugin answer a page of that plugin alone, with
// the permission filter of the list, and a disabled plugin reads as absent.
func TestTheWidgetsOfOnePluginAnswerAPage(t *testing.T) {
	t.Parallel()

	bench := widgetsUnderTest(t, widget.Deadline,
		staticWidget(pluginP, widgetOpen, thingsRead, widgetOpen),
		staticWidget(pluginP, widgetClosed, thingsWrite, widgetClosed),
		staticWidget(pluginQ, "of-q", thingsRead, "q"),
	)
	bench.enable(t, bench.h, pluginP)
	bench.enable(t, bench.h, pluginQ)

	response := bench.call(
		t,
		bench.gio,
		http.MethodGet,
		bench.inH("/plugins/"+pluginP+"/widgets"),
		nil,
		"",
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	keys := make([]string, 0, 1)
	for _, item := range items(t, response) {
		keys = append(keys, stringOf(t, item[memberKey]))
	}

	require.Equal(t, []string{widgetOpen}, keys)

	bench.disableAll(t)

	response = bench.call(
		t,
		bench.gio,
		http.MethodGet,
		bench.inH("/plugins/"+pluginP+"/widgets"),
		nil,
		"",
	)
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
	require.Equal(t, typeNotFound, decode(t, response)[memberType])
}

// WIDGET-SC-006: A viewer sees and reads the widget of things.read alone, and an
// editor sees both.
func TestTheListOmitsAWidgetThatTheRoleCannotRead(t *testing.T) {
	t.Parallel()

	bench := widgetsUnderTest(t, widget.Deadline,
		staticWidget(pluginP, widgetOpen, thingsRead, widgetOpen),
		staticWidget(pluginP, widgetClosed, thingsWrite, widgetClosed),
	)
	bench.enable(t, bench.h, pluginP)

	require.Equal(t, []string{widgetOpen}, bench.widgetsIn(t, bench.gio, bench.h))
	require.Equal(t, widgetOpen, valueOf(t, bench.read(t, bench.gio, bench.h, widgetOpen)))

	response := bench.readRaw(t, bench.gio, bench.h, pluginP, widgetClosed)
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
	require.Equal(t, "/problems/http/permission-denied", decode(t, response)[memberType])

	require.Equal(t, []string{widgetOpen, widgetClosed}, bench.widgetsIn(t, bench.beka, bench.h))
}

// WIDGET-SC-007: The list follows the plugin identifiers, then the order of the
// declaration, on every read.
func TestTheListKeepsOneOrder(t *testing.T) {
	t.Parallel()

	bench := widgetsUnderTest(t, widget.Deadline,
		staticWidget(pluginQ, "z1", thingsRead, "z1"),
		staticWidget(pluginP, "a2", thingsRead, "a2"),
		staticWidget(pluginP, "a1", thingsRead, "a1"),
	)
	bench.enable(t, bench.h, pluginQ)
	bench.enable(t, bench.h, pluginP)

	for range 10 {
		require.Equal(t, []string{"a2", "a1", "z1"}, bench.widgetsIn(t, bench.ana, bench.h))
	}
}

// WIDGET-SC-009: Load runs as the acting user in the acting workspace, and a person
// reads no widget of a workspace that they are not a member of.
func TestAWidgetReadsAsTheActingUserInTheActingWorkspace(t *testing.T) {
	t.Parallel()

	bench := widgetsUnderTest(t, widget.Deadline, loadingWidget(pluginP, "who", thingsRead,
		func(ctx context.Context) (pluginapi.WidgetAnswer, error) {
			return pluginapi.StatAnswer(pluginapi.StatData{
				Value:   pluginapi.ActingUserFromContext(ctx),
				Caption: pluginapi.ActingWorkspaceFromContext(ctx),
			}), nil
		}))

	workspaceV := bench.setup.create(t, bench.ana, "V")
	bench.setup.add(t, bench.ana, workspaceV, bench.nino, roleViewer)
	bench.enable(t, bench.h, pluginP)
	bench.enable(t, workspaceV, pluginP)

	inH := bench.read(t, bench.ana, bench.h, "who")
	inV := bench.read(t, bench.ana, workspaceV, "who")

	assert.Equal(t, stateReady, inH[memberState])
	assert.Equal(t, map[string]any{memberValue: bench.ana.id, "caption": bench.h}, inH[memberStat])
	assert.Equal(
		t,
		map[string]any{memberValue: bench.ana.id, "caption": workspaceV},
		inV[memberStat],
	)

	response := bench.readRaw(t, bench.nino, bench.h, pluginP, "who")
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
	require.Equal(t, typeNotFound, decode(t, response)[memberType])
}

// WIDGET-SC-011: A Load that fails or panics answers 500, the hub keeps serving, and
// a log record names the plugin, the key, and the cause.
func TestAFailedWidgetAnswersInternal(t *testing.T) {
	t.Parallel()

	bench := widgetsUnderTest(t, widget.Deadline,
		loadingWidget(pluginP, "failing", thingsRead,
			func(context.Context) (pluginapi.WidgetAnswer, error) {
				return pluginapi.WidgetAnswer{}, errParkingDown
			}),
		loadingWidget(pluginP, "panicking", thingsRead,
			func(context.Context) (pluginapi.WidgetAnswer, error) {
				panic("the plugin broke")
			}),
		staticWidget(pluginP, "healthy", thingsRead, "fine"),
	)
	bench.enable(t, bench.h, pluginP)

	for _, key := range []string{"failing", "panicking"} {
		response := bench.readRaw(t, bench.ana, bench.h, pluginP, key)
		require.Equal(t, http.StatusInternalServerError, response.Code, response.Body.String())
		require.Equal(t, "/problems/http/internal", decode(t, response)[memberType])
	}

	require.Equal(t, "fine", valueOf(t, bench.read(t, bench.ana, bench.h, "healthy")))

	logged := bench.lines.String()
	assert.Contains(t, logged, pluginP)
	assert.Contains(t, logged, "failing")
	assert.Contains(t, logged, "panicking")
	assert.Contains(t, logged, errParkingDown.Error())
	assert.Contains(t, logged, "the plugin broke")
}

// WIDGET-SC-013: A plugin that misses the deadline answers 504 with the type of the hub.
func TestASlowWidgetAnswersTheTimeout(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	t.Cleanup(func() { close(release) })

	bench := widgetsUnderTest(t, 50*time.Millisecond, loadingWidget(pluginP, "slow", thingsRead,
		func(context.Context) (pluginapi.WidgetAnswer, error) {
			<-release

			return pluginapi.StatAnswer(pluginapi.StatData{Value: "late"}), nil
		}))
	bench.enable(t, bench.h, pluginP)

	response := bench.readRaw(t, bench.ana, bench.h, pluginP, "slow")
	require.Equal(t, http.StatusGatewayTimeout, response.Code, response.Body.String())
	require.Equal(t, "/problems/hub/widget-timeout", decode(t, response)[memberType])
}

// WIDGET-DD-006: The route of one widget mounts beside the settings of a plugin on
// one router.
func TestTheWidgetRoutesMountBesideTheSettingsOfAPlugin(t *testing.T) {
	t.Parallel()

	bench := widgetsUnderTest(t, widget.Deadline)

	require.NotPanics(t, func() {
		handler.NewPlugin(
			slog.New(slog.DiscardHandler),
			workspaceVerifier(t, bench.provider),
			auth.NewResolver(bench.database),
			loadedCatalog(t), registry.NewUIRegistry(),
			&stubSettings{}, noIdempotency{},
			handler.WorkspaceAccess{DB: bench.database, Authorizer: widgetAuthorizer(t)},
			bench.database,
		).Register(bench.router)
	})
}
