package handler_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/gwp/db"
	"github.com/abgeo/maroid/plugins/gwp/handler"
	"github.com/abgeo/maroid/plugins/gwp/model"
	"github.com/abgeo/maroid/plugins/gwp/permission"
	"github.com/abgeo/maroid/plugins/gwp/service"
)

const (
	pathReadingList = "/readings"
	externalURL     = "https://hub.example.ge"
)

// readingPage is the answer of one page of readings.
type readingPage struct {
	Items []readingItem `json:"items"`
	Next  *string       `json:"next"`
	Prev  *string       `json:"prev"`
}

type readingItem struct {
	ID           string   `json:"id"`
	SerialNumber string   `json:"serial_number"`
	ReadingDate  string   `json:"reading_date"`
	Value        float64  `json:"value"`
	Usage        *float64 `json:"usage"`
}

// readingsUnderTest migrates a database, and answers the store of workspace W and
// the route of the readings over it, as the hub mounts the route.
func readingsUnderTest(t *testing.T) (*service.Store, http.Handler) {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(`CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
		INSERT INTO public.workspaces (id) VALUES ('` + workspaceW + `');`)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID("dev.maroid.gwp")
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	store := service.NewStore(pluginapi.NewPluginDB(instance.DB, identifier))
	routes := handler.NewReadingHandler(slog.New(slog.DiscardHandler), store).Routes()
	require.Len(t, routes, 1)

	actingInWorkspace := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(
				w,
				r.WithContext(pluginapi.ContextWithActingWorkspace(r.Context(), workspaceW)),
			)
		})
	}

	return store, address.Middleware(externalURL)(actingInWorkspace(routes[0].Handler))
}

// monthly answers a reading of the meter on the 15th of each month from the first
// month on, with a value that grows by step each month.
func monthly(
	customer, serial string,
	first time.Time,
	months int,
	start, step float64,
) []model.Reading {
	readings := make([]model.Reading, 0, months)

	for month := range months {
		readings = append(readings, model.Reading{
			CustomerNumber: customer,
			SerialNumber:   serial,
			ReadingDate:    first.AddDate(0, month, 0),
			Value:          start + step*float64(month),
		})
	}

	return readings
}

func day(year int, month time.Month) time.Time {
	return time.Date(year, month, 15, 0, 0, 0, 0, time.UTC)
}

func get(t *testing.T, route http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	route.ServeHTTP(recorder, request)

	return recorder
}

func pageOf(t *testing.T, recorder *httptest.ResponseRecorder) readingPage {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var answered readingPage
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answered))

	return answered
}

// follow answers the target of a link of a page, relative to the hub.
func follow(t *testing.T, link *string) string {
	t.Helper()

	require.NotNil(t, link)

	parsed, err := url.Parse(*link)
	require.NoError(t, err)

	return parsed.RequestURI()
}

// twoMetersOverTwoYears assigns C1, and stores a reading of meter A, then one of
// meter B, on the 15th of each month from October 2024 to September 2026. A grows by
// 10 a month, and B by 2.
func twoMetersOverTwoYears(t *testing.T, store *service.Store) {
	t.Helper()

	ctx := actingInW()
	require.NoError(t, store.Assign(ctx, customerC1))

	for month := range 24 {
		date := day(2024, time.October).AddDate(0, month, 0)
		require.NoError(t, store.InsertReadings(ctx, []model.Reading{
			{
				CustomerNumber: customerC1,
				SerialNumber:   "A",
				ReadingDate:    date,
				Value:          100 + 10*float64(month),
			},
			{
				CustomerNumber: customerC1,
				SerialNumber:   "B",
				ReadingDate:    date,
				Value:          50 + 2*float64(month),
			},
		}))
	}
}

// GWPACC-SC-025: 48 readings answer pages of 20, 20, and 8, the latest date first,
// with the links of each page, and the usage of each reading but the earliest of a meter.
func TestTheReadingsArePagedTheLatestFirst(t *testing.T) {
	t.Parallel()

	store, route := readingsUnderTest(t)
	twoMetersOverTwoYears(t, store)

	first := pageOf(t, get(t, route, pathReadingList))
	require.Len(t, first.Items, 20)
	require.NotNil(t, first.Next)
	require.Nil(t, first.Prev)
	require.Equal(t, "2026-09-15", first.Items[0].ReadingDate)
	require.Equal(
		t,
		"B",
		first.Items[0].SerialNumber,
		"B came after A on one date, so its identifier is greater",
	)
	require.Equal(t, "A", first.Items[1].SerialNumber)
	require.InDelta(t, 2.0, *first.Items[0].Usage, 0)
	require.InDelta(t, 10.0, *first.Items[1].Usage, 0)

	second := pageOf(t, get(t, route, follow(t, first.Next)))
	require.Len(t, second.Items, 20)
	require.NotNil(t, second.Next)
	require.NotNil(t, second.Prev)

	last := pageOf(t, get(t, route, follow(t, second.Next)))
	require.Len(t, last.Items, 8)
	require.Nil(t, last.Next)
	require.NotNil(t, last.Prev)

	all := append(append(first.Items, second.Items...), last.Items...)
	for i, item := range all {
		if i > 0 {
			require.GreaterOrEqual(t, all[i-1].ReadingDate, item.ReadingDate)
		}

		if item.ReadingDate == "2024-10-15" {
			require.Nil(t, item.Usage, "the earliest reading of a meter has no usage")
		} else {
			require.NotNil(t, item.Usage, item.ReadingDate)
		}
	}
}

// GWPACC-SC-026: The page holds the readings of the assigned customer alone.
func TestTheReadingsOfAnEarlierCustomerStayHidden(t *testing.T) {
	t.Parallel()

	store, route := readingsUnderTest(t)
	ctx := actingInW()

	require.NoError(
		t,
		store.InsertReadings(ctx, monthly(customerC1, "A", day(2026, time.July), 3, 100, 5)),
	)
	require.NoError(
		t,
		store.InsertReadings(ctx, monthly(customerC2, "B", day(2026, time.July), 3, 10, 1)),
	)
	require.NoError(t, store.Assign(ctx, customerC2))

	answered := pageOf(t, get(t, route, pathReadingList))

	require.Len(t, answered.Items, 3)

	for _, item := range answered.Items {
		require.Equal(t, "B", item.SerialNumber)
	}
}

// GWPACC-SC-027: The filter from keeps the readings of the date and after, the usage
// of the first counts from the reading before the date, and a malformed date is refused.
func TestTheReadingsFilterFromADate(t *testing.T) {
	t.Parallel()

	store, route := readingsUnderTest(t)
	ctx := actingInW()

	require.NoError(t, store.Assign(ctx, customerC1))
	require.NoError(t, store.InsertReadings(ctx, []model.Reading{
		{
			CustomerNumber: customerC1,
			SerialNumber:   "A",
			ReadingDate:    day(2025, time.September),
			Value:          100,
		},
		{
			CustomerNumber: customerC1,
			SerialNumber:   "A",
			ReadingDate:    day(2025, time.October),
			Value:          110,
		},
		{
			CustomerNumber: customerC1,
			SerialNumber:   "A",
			ReadingDate:    day(2026, time.September),
			Value:          200,
		},
	}))

	answered := pageOf(t, get(t, route, pathReadingList+"?from=2025-10-12"))

	require.Len(t, answered.Items, 2)
	require.Equal(t, "2026-09-15", answered.Items[0].ReadingDate)
	require.InDelta(t, 90.0, *answered.Items[0].Usage, 0)
	require.Equal(t, "2025-10-15", answered.Items[1].ReadingDate)
	require.InDelta(t, 10.0, *answered.Items[1].Usage, 0)

	refused := get(t, route, pathReadingList+"?from=12-10-2025")

	requireProblem(t, refused, http.StatusBadRequest, "/problems/http/request-invalid")
	require.Contains(t, decode(t, refused)["detail"], "from")
}

// GWPACC-SC-028: With no assigned customer the route answers 409 not-assigned.
func TestTheReadingsAskForAnAssignment(t *testing.T) {
	t.Parallel()

	_, route := readingsUnderTest(t)

	requireProblem(
		t,
		get(t, route, pathReadingList),
		http.StatusConflict,
		problemPrefix+"not-assigned",
	)
}

// GWPACC-SC-029: An assigned customer with no reading answers an empty page with no links.
func TestTheReadingsOfACustomerWithNoReading(t *testing.T) {
	t.Parallel()

	store, route := readingsUnderTest(t)
	require.NoError(t, store.Assign(actingInW(), customerC1))

	answered := pageOf(t, get(t, route, pathReadingList))

	require.Empty(t, answered.Items)
	require.NotNil(t, answered.Items, "an empty page answers an empty array, not null")
	require.Nil(t, answered.Next)
	require.Nil(t, answered.Prev)
}

// GWPACC-SC-031: GET /readings names account.read.
func TestTheReadingsRouteNamesItsPermission(t *testing.T) {
	t.Parallel()

	routes := handler.NewReadingHandler(slog.New(slog.DiscardHandler), nil).Routes()

	require.Len(t, routes, 1)
	require.Equal(t, http.MethodGet, routes[0].Method)
	require.Equal(t, pathReadingList, routes[0].Pattern)
	require.Equal(t, permission.AccountRead, routes[0].Permission)
}
