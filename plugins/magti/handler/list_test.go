package handler_test

import (
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/magti/db"
	"github.com/abgeo/maroid/plugins/magti/handler"
	"github.com/abgeo/maroid/plugins/magti/internal/fakeportal"
	"github.com/abgeo/maroid/plugins/magti/internal/fakes"
	"github.com/abgeo/maroid/plugins/magti/model"
	"github.com/abgeo/maroid/plugins/magti/service"
)

const (
	pathTraffic      = "/traffic-days"
	pathMoneyEntries = "/money-entries"
	pathMonthTotals  = "/monthly-money-totals"
)

type pageAnswer[T any] struct {
	Items []T     `json:"items"`
	Next  *string `json:"next"`
}

type trafficAnswer struct {
	Day         string   `json:"day"`
	DataMB      *float64 `json:"data_mb"`
	SMSSent     *int     `json:"sms_sent"`
	SMSReceived *int     `json:"sms_received"`
}

type entryAnswer struct {
	ID         string      `json:"id"`
	OccurredAt string      `json:"occurred_at"`
	Direction  string      `json:"direction"`
	Amount     moneyAnswer `json:"amount"`
}

type totalAnswer struct {
	Month   string      `json:"month"`
	TopUps  moneyAnswer `json:"top_ups"`
	Charges moneyAnswer `json:"charges"`
}

func readTotals(query string) request {
	return request{method: http.MethodGet, pattern: pathMonthTotals, query: query}
}

// MAGTIACC-SC-026: The three reads answer 409 with no assignment or with a home line, an
// empty page with no row, and call no portal.
func TestTheReadsNeedAMobileSubscriber(t *testing.T) {
	t.Parallel()

	reads := []request{
		{method: http.MethodGet, pattern: pathTraffic, query: "from=2026-11-01&to=2026-11-24"},
		{method: http.MethodGet, pattern: pathMoneyEntries},
		readTotals("from=2025-12"),
	}

	for store, slug := range map[*fakes.Store]string{
		fakes.NewStore("", ""): "not-assigned",
		fakes.NewStore(fakeportal.HomeLine, model.SubscriberKindHomeLine): "home-line",
	} {
		rig := newRouteRig(t, fakeportal.SessionCookie, store)

		for _, read := range reads {
			recorder := rig.do(t, read)
			assert.Equal(t, http.StatusConflict, recorder.Code)
			assert.Equal(t, problemPrefix+slug, problemTypeOf(t, recorder))
		}

		assert.Empty(t, rig.Portal.Calls())
	}

	rig := newRouteRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))

	assert.Empty(t, decodeAs[pageAnswer[trafficAnswer]](t, rig.do(t, reads[0])).Items)
	assert.Empty(t, decodeAs[pageAnswer[entryAnswer]](t, rig.do(t, reads[1])).Items)

	totals := decodeAs[pageAnswer[totalAnswer]](t, rig.do(t, reads[2])).Items
	require.Len(t, totals, 12)

	for _, total := range totals {
		assert.Zero(t, total.TopUps.Amount)
		assert.Zero(t, total.Charges.Amount)
	}

	assert.Empty(t, rig.Portal.Calls())
}

// MAGTIACC-SC-024, MAGTIACC-SC-025: A period of more than a year, a sort field that the
// route does not take, and a first month more than 11 months back are refused.
func TestTheReadsRefuseTheirLimits(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeportal.SessionCookie,
		fakes.NewStore(fakeportal.Mobile, model.SubscriberKindMobile))

	for _, call := range []request{
		{method: http.MethodGet, pattern: pathTraffic, query: "from=2025-01-01&to=2026-11-24"},
		{method: http.MethodGet, pattern: pathTraffic, query: "to=2026-11-24"},
		readTotals("from=2025-01"),
		readTotals(""),
	} {
		recorder := rig.do(t, call)
		assert.Equal(t, http.StatusBadRequest, recorder.Code, call.query)
		assert.Equal(t, "/problems/http/request-invalid", problemTypeOf(t, recorder))
	}

	recorder := rig.do(t, request{
		method:  http.MethodGet,
		pattern: pathTraffic,
		query:   "from=2026-11-01&to=2026-11-24&sort=data_mb",
	})
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "data_mb")
}

// listRig serves the reads over the store of a database.
type listRig struct {
	store  *service.Store
	routes []pluginapi.Route
}

func newListRig(t *testing.T) *listRig {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(`
		CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
		INSERT INTO public.workspaces (id) VALUES ('` + fakes.WorkspaceW + `');`)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID("dev.maroid.magti")
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	store := service.NewStore(testdb.PluginDB(instance, identifier))
	logger := slog.New(slog.DiscardHandler)

	_, err = store.Assign(
		fakes.Acting(t.Context()), fakeportal.Mobile, model.SubscriberKindMobile, nil,
	)
	require.NoError(t, err)

	return &listRig{
		store: store,
		routes: slices.Concat(
			handler.NewTrafficHandler(logger, store).Routes(),
			handler.NewMoneyHandler(logger, store, now).Routes(),
		),
	}
}

func (r *listRig) follow(t *testing.T, call request, next string) request {
	t.Helper()

	link, err := url.Parse(next)
	require.NoError(t, err)

	call.query = link.RawQuery

	return call
}

// MAGTIACC-SC-024: The pages of the traffic hold 20 and 9 rows, the earliest day first,
// and a day with no row is absent.
func TestTheTrafficPagesRunFromTheEarliestDay(t *testing.T) {
	t.Parallel()

	rig := newListRig(t)
	first := model.Day{Year: 2026, Month: time.October, Day: 26}

	var days []model.TrafficDay

	for offset := range 30 {
		day := first.AddDays(offset)
		if day.String() == "2026-11-02" {
			continue
		}

		days = append(days, model.TrafficDay{
			SubscriberNumber: fakeportal.Mobile,
			Day:              day.Date(),
			DataMB:           new(float64(offset)),
		})
	}

	_, err := rig.store.SaveCollection(fakes.Acting(t.Context()), days, nil)
	require.NoError(t, err)

	call := request{
		method:  http.MethodGet,
		pattern: pathTraffic,
		query:   "from=2026-10-26&to=2026-11-24&limit=20",
	}

	firstPage := decodeAs[pageAnswer[trafficAnswer]](t, serve(t, rig.routes, call))
	require.Len(t, firstPage.Items, 20)
	assert.Equal(t, "2026-10-26", firstPage.Items[0].Day)
	assert.Nil(t, firstPage.Items[0].SMSSent)
	require.NotNil(t, firstPage.Next)

	second := decodeAs[pageAnswer[trafficAnswer]](t,
		serve(t, rig.routes, rig.follow(t, call, *firstPage.Next)))
	assert.Len(t, second.Items, 9)

	for _, row := range append(firstPage.Items, second.Items...) {
		assert.NotEqual(t, "2026-11-02", row.Day)
	}

	call.query += "&sort=-day"
	latest := decodeAs[pageAnswer[trafficAnswer]](t, serve(t, rig.routes, call))
	assert.Equal(t, "2026-11-24", latest.Items[0].Day)
}

// MAGTIACC-SC-025: The entries come the latest first, and the months answer 12 items
// with the sums of each month and zero for an empty one.
func TestTheMoneyAndItsTotals(t *testing.T) {
	t.Parallel()

	rig := newListRig(t)
	entries := make([]model.MoneyEntry, 0, 5)

	for _, raw := range []struct {
		moment    string
		direction model.Direction
		kind      string
		amount    model.Tetri
	}{
		{"13/11/2026 01:50:48", model.DirectionIn, topUp, 1730},
		{"13/11/2026 01:50:49", model.DirectionOut, "Plan", 200},
		{"13/11/2026 01:50:49", model.DirectionOut, "SMS", 200},
		{"15/11/2026 10:00:00", model.DirectionIn, topUp, 200},
		{"05/11/2026 10:00:00", model.DirectionIn, topUp, 490},
	} {
		moment, err := time.ParseInLocation("02/01/2006 15:04:05", raw.moment, model.Tbilisi())
		require.NoError(t, err)

		entries = append(entries, model.MoneyEntry{
			SubscriberNumber: fakeportal.Mobile,
			OccurredAt:       moment,
			Direction:        raw.direction,
			Kind:             raw.kind,
			Amount:           raw.amount,
		})
	}

	_, err := rig.store.SaveCollection(fakes.Acting(t.Context()), nil, entries)
	require.NoError(t, err)

	list := decodeAs[pageAnswer[entryAnswer]](t, serve(t, rig.routes, request{
		method: http.MethodGet, pattern: pathMoneyEntries,
	}))
	require.Len(t, list.Items, 5)
	assert.Equal(t, "2026-11-15T06:00:00Z", list.Items[0].OccurredAt)
	assert.Equal(t, "2026-11-05T06:00:00Z", list.Items[4].OccurredAt)

	totals := decodeAs[pageAnswer[totalAnswer]](t, serve(t, rig.routes, readTotals("from=2025-12")))
	require.Len(t, totals.Items, 12)
	assert.Nil(t, totals.Next)
	assert.Equal(t, totalAnswer{
		Month:   "2026-11",
		TopUps:  moneyAnswer{Amount: 24.20, Currency: lari},
		Charges: moneyAnswer{Amount: 4.00, Currency: lari},
	}, totals.Items[0])
	assert.Equal(t, "2026-10", totals.Items[1].Month)
	assert.Zero(t, totals.Items[1].TopUps.Amount)

	ascending := decodeAs[pageAnswer[totalAnswer]](t,
		serve(t, rig.routes, readTotals("from=2025-12&sort=month")))
	assert.Equal(t, "2025-12", ascending.Items[0].Month)
}
