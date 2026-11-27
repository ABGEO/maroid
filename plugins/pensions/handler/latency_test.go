package handler_test

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/pensions/db"
	"github.com/abgeo/maroid/plugins/pensions/handler"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

const (
	userB             = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e60"
	heldContributions = 2000
	heldYears         = 10
	readsPerFilter    = 20
	latencyLimit      = 300 * time.Millisecond
)

// heldEmployers answers the codes of the four employers of the contributions.
func heldEmployers() []string {
	return []string{"204567890", "405112233", "211998877", "302445566"}
}

// heldStore starts a database with the migrations of the plugin, and the users that a
// row of a user references.
func heldStore(t *testing.T) *service.Store {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(`
		CREATE TABLE public.users (id UUID NOT NULL PRIMARY KEY);
		INSERT INTO public.users (id) VALUES ('` + userA + `'), ('` + userB + `');`)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID("dev.maroid.pensions")
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	return service.NewStore(testdb.PluginDB(instance, identifier))
}

// tenYearsOf answers the contributions of four employers, spread evenly over ten years
// that end in October 2026.
func tenYearsOf() []model.Contribution {
	first := time.Date(2016, time.November, 1, 0, 0, 0, 0, time.UTC)
	span := heldYears * 365
	rows := make([]model.Contribution, 0, heldContributions)
	employers := heldEmployers()

	for i := range heldContributions {
		code := employers[i%len(employers)]
		rows = append(rows, model.Contribution{
			BasisID:          uuid.New(),
			Source:           "დასაქმებული",
			ContributionType: "ჩვეულებრივი",
			PaidOn:           first.AddDate(0, 0, i*span/heldContributions),
			GrossSalary:      5000,
			Amount:           new(100.0),
			EmployerCode:     &code,
			Employer:         &model.Employer{Code: code, Name: "შპს " + code},
		})
	}

	return rows
}

// servedAs answers a server of the contribution routes that acts as the user, as the hub
// serves them.
func servedAs(t *testing.T, store *service.Store, user string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	for _, route := range handler.NewContributionHandler(slog.New(slog.DiscardHandler), store).Routes() {
		mux.HandleFunc(route.Method+" "+route.Pattern, route.Handler)
	}

	acting := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(pluginapi.ContextWithActingUser(r.Context(), user)))
	})

	server := httptest.NewServer(address.Middleware("https://maroid.test")(acting))
	t.Cleanup(server.Close)

	return server
}

// p95 answers the 95th percentile of the time from the request to the last byte.
func p95(t *testing.T, target string) time.Duration {
	t.Helper()

	durations := make([]time.Duration, 0, readsPerFilter)

	for range readsPerFilter {
		started := time.Now()

		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
		require.NoError(t, err)

		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)

		_, err = io.ReadAll(response.Body)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		require.Equal(t, http.StatusOK, response.StatusCode, target)

		durations = append(durations, time.Since(started))
	}

	slices.Sort(durations)

	return durations[(len(durations)*95+99)/100-1]
}

// PENSION-SC-045: The page and the total of the contributions answer within 300
// milliseconds at the 95th percentile, with 2,000 contributions of each of two users.
func TestTheContributionsAnswerWithinTheLimit(t *testing.T) {
	t.Parallel()

	store := heldStore(t)

	for _, user := range []string{userA, userB} {
		ctx := pluginapi.ContextWithActingUser(t.Context(), user)
		require.NoError(t, store.Save(ctx, model.Snapshot{Contributions: tenYearsOf()}))
	}

	server := servedAs(t, store, userA)
	employer := heldEmployers()[0]
	filters := map[string]string{
		"none":     "",
		"employer": "employer=" + employer,
		"period":   "from=2020-01-01&to=2022-12-31",
		"all":      "employer=" + employer + "&from=2020-01-01&to=2022-12-31",
	}

	for _, path := range []string{"/contributions", "/contributions/total"} {
		for name, query := range filters {
			measured := p95(t, fmt.Sprintf("%s%s?%s", server.URL, path, query))

			t.Logf("%s, filter %s: p95 %s", path, name, measured)
			assert.Less(t, measured, latencyLimit, "%s, filter %s", path, name)
		}
	}
}
