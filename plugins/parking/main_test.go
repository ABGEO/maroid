package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/db"
	"github.com/abgeo/maroid/plugins/parking/permission"
	"github.com/abgeo/maroid/plugins/parking/service"
)

// PARKACC-SC-001: The settings hold the username as free text and the password as a
// secret, both of the user and both required, and no token or vehicle.
func TestTheSettingsHoldTheCredentialsAlone(t *testing.T) {
	t.Parallel()

	model, err := (&ParkingPlugin{}).SettingsModel()
	require.NoError(t, err)

	settings := reflect.TypeOf(model)
	require.Equal(t, reflect.TypeFor[config.UserSettings](), settings)
	require.Equal(t, 2, settings.NumField())

	username, found := settings.FieldByName("Username")
	require.True(t, found)
	assert.Equal(t, "username", username.Tag.Get("json"))
	assert.Equal(t, "title=Username,required", username.Tag.Get("jsonschema"))
	assert.Equal(t, "x-maroid-scope=user", username.Tag.Get("jsonschema_extras"))

	password, found := settings.FieldByName("Password")
	require.True(t, found)
	assert.Equal(t, "password", password.Tag.Get("json"))
	assert.Equal(
		t,
		"title=Password,format=password,writeOnly=true,required",
		password.Tag.Get("jsonschema"),
	)
	assert.Equal(t, "x-maroid-scope=user", password.Tag.Get("jsonschema_extras"))
}

// PARKACC-SC-022: account.connect has viewer as its lowest role.
func TestAccountConnectIsOpenToAViewer(t *testing.T) {
	t.Parallel()

	permissions, err := (&ParkingPlugin{}).Permissions()
	require.NoError(t, err)

	for _, declared := range permissions {
		if declared.Name == permission.AccountConnect {
			assert.Equal(t, pluginapi.RoleViewer, declared.Lowest)

			return
		}
	}

	t.Fatal("the plugin declares no account.connect")
}

// PARKHIST-SC-001: An empty config gives one job of each user, each hour at minute 15.
func TestTheHistoryCollectorRunsEachHour(t *testing.T) {
	t.Parallel()

	cfg := new(config.Config)
	require.NoError(t, pluginconfig.DecodeAndValidateConfig(map[string]any{}, cfg))

	jobs, err := (&ParkingPlugin{config: cfg, logger: slog.New(slog.DiscardHandler)}).CronJobs()
	require.NoError(t, err)

	require.Len(t, jobs, 1)
	assert.Equal(t, pluginapi.CronJobMeta{
		ID:       "history_collector",
		Schedule: "15 * * * *",
		Scope:    pluginapi.CronScopePerUser,
	}, jobs[0].Meta())
	assert.Equal(t, 5, cfg.CollectorPages)
}

// heldToken holds a session token for every user, so a call reaches the parking service.
type heldToken struct {
	pluginapi.PluginSecrets
}

func (heldToken) Get(context.Context, pluginapi.SecretScope, string) (string, error) {
	return "T1", nil
}

// historyPaths answers the routes of the history pages.
func historyPaths() []string {
	return []string{
		"/sessions", "/sessions/total", "/plates", "/lot-usages",
		"/transactions", "/transactions/total", "/charge-months",
	}
}

// PARKHIST-SC-032: history.read has the lowest role viewer, and each history route
// names it.
func TestEachHistoryRouteNamesHistoryRead(t *testing.T) {
	t.Parallel()

	plugin := &ParkingPlugin{logger: slog.New(slog.DiscardHandler)}

	permissions, err := plugin.Permissions()
	require.NoError(t, err)
	assert.Contains(t, permissions, pluginapi.Permission{
		Name:        permission.HistoryRead,
		Description: "Read your own past sessions and transactions",
		Lowest:      pluginapi.RoleViewer,
	})

	routes, err := plugin.Routes()
	require.NoError(t, err)

	named := map[string]string{}
	for _, route := range routes {
		named[route.Pattern] = route.Permission
	}

	for _, path := range historyPaths() {
		assert.Equal(t, permission.HistoryRead, named[path], path)
	}
}

// PARKHIST-SC-031: Each history route answers from the database while every call of the
// parking service fails, and makes no call.
func TestTheHistoryRoutesCallNoService(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	parking := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(parking.Close)

	instance := testdb.Start(t)
	_, err := instance.DB.Exec(`CREATE TABLE public.users (id UUID NOT NULL PRIMARY KEY);`)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "dev_maroid_parking", migrations)

	cfg := &config.Config{BaseURL: parking.URL}
	client := service.NewAPIClient(cfg, heldToken{})
	plugin := &ParkingPlugin{
		config:       cfg,
		logger:       slog.New(slog.DiscardHandler),
		apiClientSvc: client,
		history:      service.NewHistoryStore(testdb.PluginDB(instance, Meta.ID)),
	}

	routes, err := plugin.Routes()
	require.NoError(t, err)

	ctx := pluginapi.ContextWithActingUser(
		context.Background(),
		"0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e77",
	)

	for _, route := range routes {
		if !slices.Contains(historyPaths(), route.Pattern) {
			continue
		}

		request := httptest.NewRequestWithContext(ctx, http.MethodGet, route.Pattern, nil)
		recorder := httptest.NewRecorder()
		address.Middleware("https://maroid.test")(route.Handler).ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code, route.Pattern)
	}

	assert.Zero(t, calls.Load())
}
