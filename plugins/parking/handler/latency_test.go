package handler_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/plugins/parking/model"
)

const (
	heldRows       = 5000
	heldYears      = 10
	readsPerFilter = 20
	latencyLimit   = 300 * time.Millisecond
)

// tenYearsOfHistory answers the sessions of three plates and the transactions of the four
// kinds, spread evenly over ten years that end in November 2026. Each tenth session is
// free.
func tenYearsOfHistory() ([]model.Session, []model.Transaction) {
	first := time.Date(2016, time.November, 1, 8, 0, 0, 0, time.UTC)
	step := heldYears * 365 * 24 * time.Hour / heldRows
	plates := []string{plateA, plateB, "EF456GH"}
	kinds := []model.TransactionKind{
		model.TransactionKindParkingPayment, model.TransactionKindTopUp,
		model.TransactionKindPermitPurchase, model.TransactionKindOther,
	}

	sessions := make([]model.Session, 0, heldRows)
	transactions := make([]model.Transaction, 0, heldRows)

	for i := range heldRows {
		moment := first.Add(time.Duration(i) * step)

		amount := 0.55
		if i%10 == 0 {
			amount = 0
		}

		sessions = append(
			sessions,
			sessionAt(int64(i+1), lotA1637, plates[i%len(plates)], moment, amount),
		)
		transactions = append(transactions,
			transactionAt(int64(i+1), kinds[i%len(kinds)], moment.Add(time.Hour), 0.55, 9.08))
	}

	return sessions, transactions
}

// servedAs answers a server of the history routes that acts as the user in Asia/Tbilisi,
// as the hub serves them.
func servedAs(t *testing.T, rig *historyRig, user string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	for _, route := range rig.routes {
		mux.HandleFunc(route.Method+" "+route.Pattern, route.Handler)
	}

	acting := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(actingFrom(r.Context(), t, user)))
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

// PARKHIST-SC-033: Each list and each total answers within 300 milliseconds at the 95th
// percentile, with 5,000 sessions and 5,000 transactions of each of two users.
func TestTheHistoryAnswersWithinTheLimit(t *testing.T) {
	t.Parallel()

	rig := newHistoryRig(t)

	for _, user := range []string{userA, userB} {
		sessions, transactions := tenYearsOfHistory()
		rig.save(t, user, sessions, transactions)
	}

	server := servedAs(t, rig, userA)

	const period = "from=2020-01-01&to=2022-12-31"

	routes := map[string]map[string]string{
		"/sessions": {
			"none": "", "dates": period, "plate": "plate=" + plateA, parameterFree: "free=false",
			"all": period + "&plate=" + plateA + "&free=false",
		},
		"/transactions": {
			"none": "", "dates": period, "kind": "kind=top_up",
			"all": period + "&kind=top_up",
		},
	}

	for path, filters := range routes {
		for _, route := range []string{path, path + "/total"} {
			for name, query := range filters {
				measured := p95(t, fmt.Sprintf("%s%s?%s", server.URL, route, query))

				t.Logf("%s, filter %s: p95 %s", route, name, measured)
				assert.Less(t, measured, latencyLimit, "%s, filter %s", route, name)
			}
		}
	}
}
