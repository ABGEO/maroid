package handler_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/drivvo/config"
	"github.com/abgeo/maroid/plugins/drivvo/db"
	"github.com/abgeo/maroid/plugins/drivvo/handler"
	"github.com/abgeo/maroid/plugins/drivvo/internal/fakes"
	"github.com/abgeo/maroid/plugins/drivvo/internal/fakeservice"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/service"
)

const (
	externalURL = "https://maroid.example"
	currencyGEL = "GEL"

	hatchback = "Hatchback"
	oldSedan  = "Old sedan"
	zeta      = "Zeta"
)

// hubFixtures stands in for the core migrations of the hub: the workspaces that a scoped
// table references.
const hubFixtures = `
CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.workspaces (id) VALUES ('` + fakes.WorkspaceW + `'), ('` + fakes.WorkspaceX + `');`

// pageRig serves every route of the plugin over a car log copy in PostgreSQL, as a member
// of workspace W in Tbilisi.
type pageRig struct {
	fake     *fakeservice.Service
	secrets  *fakes.Secrets
	store    *service.Store
	figures  *service.Figures
	router   http.Handler
	vehicles map[string]string
}

func newPageRig(t *testing.T, snapshot model.Snapshot) *pageRig {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID("dev.maroid.drivvo")
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	fake := fakeservice.New(t)
	client := service.NewAPIClient(&config.Config{
		BaseURL:        fake.URL(),
		RequestTimeout: 5 * time.Second,
		AppVersion:     "23",
		AppVersionName: "14.2.3",
	}, clock)
	secrets := fakes.NewSecrets().
		WithSession(fakes.WorkspaceW, fakeservice.Token, clock().Add(time.Hour))
	account := service.NewAccount(client, fakes.Credentials(), secrets, clock)
	store := service.NewStore(testdb.PluginDB(instance, identifier))
	figures := service.NewFigures(store, clock)

	rig := &pageRig{fake: fake, secrets: secrets, store: store, figures: figures}
	rig.replace(t, snapshot)

	logger := slog.New(slog.DiscardHandler)
	router := chi.NewRouter()

	for _, route := range slices.Concat(
		handler.NewConnectionHandler(logger, service.NewConnections(account, client)).Routes(),
		handler.NewCarLogHandler(logger, figures).Routes(),
	) {
		router.Method(route.Method, route.Pattern, route.Handler)
	}

	rig.router = address.Middleware(externalURL)(router)
	rig.vehicles = rig.vehicleIDs(t)

	return rig
}

func memberContext(ctx context.Context) context.Context {
	return pluginapi.ContextWithActingTimeZone(fakes.Acting(ctx, fakes.WorkspaceW), model.Tbilisi())
}

func (r *pageRig) replace(t *testing.T, snapshot model.Snapshot) {
	t.Helper()

	require.NoError(t, r.store.Replace(memberContext(t.Context()), snapshot))
}

// get serves a request of the target, a path and a query.
func (r *pageRig) get(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()

	return r.serve(t, http.MethodGet, target)
}

func (r *pageRig) serve(t *testing.T, method, target string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(memberContext(t.Context()), method, target, nil)
	recorder := httptest.NewRecorder()
	r.router.ServeHTTP(recorder, request)

	return recorder
}

// vehiclePath answers the path of the page of the vehicle of the name.
func (r *pageRig) vehiclePath(name, rest string) string {
	return "/vehicles/" + r.vehicles[name] + rest
}

func (r *pageRig) vehicleIDs(t *testing.T) map[string]string {
	t.Helper()

	ids := map[string]string{}

	for _, vehicle := range decodeAs[itemPage[vehicleItem]](t, r.get(t, "/vehicles")).Items {
		ids[vehicle.Name] = vehicle.ID
	}

	return ids
}

type vehicleItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type itemPage[T any] struct {
	Items []T     `json:"items"`
	Next  *string `json:"next"`
	Prev  *string `json:"prev"`
}

func decodeAs[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var answer T
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))

	return answer
}

// pathOf answers the path and the query of a link of a page.
func pathOf(t *testing.T, link *string) string {
	t.Helper()

	require.NotNil(t, link)

	return strings.TrimPrefix(*link, externalURL)
}

func at(month time.Month, day, hour int) time.Time {
	return time.Date(2026, month, day, hour, 0, 0, 0, time.UTC)
}

func dayOn(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func amountOf(value float64) *model.Amount {
	return new(model.AmountOf(value))
}

func fuelLine(position int, item int32, volume, amount float64, full, free bool) model.Line {
	return model.Line{
		Position: position, Item: item, Amount: model.AmountOf(amount), Fuel: true,
		UnitPrice: model.AmountOf(3), Volume: volume, FullTank: full, Free: free,
	}
}

func refuelOf(
	sourceID, vehicle int32,
	occurredAt time.Time,
	odometer, volume float64,
	full bool,
) model.LogEntry {
	return model.LogEntry{
		Kind: model.KindRefuel, SourceID: sourceID, VehicleSource: vehicle,
		OccurredAt: occurredAt, Odometer: odometer, Amount: amountOf(volume * 3),
		Lines: []model.Line{fuelLine(1, 1, volume, volume*3, full, false)},
	}
}

// pagesSnapshot holds "Hatchback" (11, principal) with an entry of each kind and four
// reminders, "Old sedan" (12, inactive) with one refuel late on 31 October, and "Zeta" (13,
// active) with 45 refuels.
func pagesSnapshot() model.Snapshot {
	snapshot := model.Snapshot{
		Settings: model.CarLogSettings{
			VolumeUnit: model.VolumeLitre, ConsumptionFormat: 4, Currency: currencyGEL,
		},
		ReferenceItems: referenceItems(),
		Vehicles:       threeVehicles(),
		LogEntries:     hatchbackEntries(),
		Reminders:      hatchbackReminders(),
	}

	snapshot.LogEntries = append(snapshot.LogEntries,
		refuelOf(601, 12, time.Date(2026, 10, 31, 21, 30, 0, 0, time.UTC), 50000, 40, true))

	for index := range int32(45) {
		snapshot.LogEntries = append(snapshot.LogEntries, refuelOf(
			700+index, 13, at(time.June, 1, 0).Add(time.Duration(index)*time.Hour),
			float64(20000+index*100), 10, true,
		))
	}

	return snapshot
}

func referenceItems() []model.ReferenceItem {
	return []model.ReferenceItem{
		{Kind: model.ReferenceFuel, SourceID: 1, Name: "Petrol"},
		{Kind: model.ReferenceFuel, SourceID: 2, Name: "LPG"},
		{Kind: model.ReferenceMaintenanceType, SourceID: 5, Name: "Oil change"},
		{Kind: model.ReferenceExpenseType, SourceID: 7, Name: "Insurance"},
		{Kind: model.ReferenceIncomeType, SourceID: 9, Name: "Ride"},
		{Kind: model.ReferenceStation, SourceID: 4, Name: "Gulf"},
		{Kind: model.ReferencePaymentMethod, SourceID: 3, Name: "Card"},
		{Kind: model.ReferencePlace, SourceID: 6, Name: "Garage"},
	}
}

func threeVehicles() []model.Vehicle {
	return []model.Vehicle{
		{SourceID: 13, Name: zeta, DistanceUnit: model.DistanceKilometre, Active: true},
		{SourceID: 12, Name: oldSedan, DistanceUnit: model.DistanceKilometre},
		{
			SourceID: 11, Name: hatchback, Make: "Make", Model: "Model",
			DistanceUnit: model.DistanceKilometre, Principal: true, Active: true,
		},
	}
}

func hatchbackEntries() []model.LogEntry {
	return []model.LogEntry{
		{
			Kind: model.KindRefuel, SourceID: 101, VehicleSource: 11,
			OccurredAt: at(time.November, 1, 8), Odometer: 10000, Amount: amountOf(90),
			Station: 4, PaymentMethod: 3,
			BatteryStartPct: new(20.0), BatteryEndPct: new(80.0), ChargeMinutes: new(int32(45)),
			Lines: []model.Line{
				fuelLine(1, 1, 30, 90, true, false),
				fuelLine(2, 2, 5, 15, false, true),
			},
		},
		refuelOf(102, 11, at(time.November, 8, 8), 10450, 30, false),
		refuelOf(103, 11, at(time.November, 15, 8), 10800, 20, true),
		{
			Kind: model.KindMaintenance, SourceID: 201, VehicleSource: 11,
			OccurredAt: at(time.November, 10, 8), Odometer: 10600, Amount: amountOf(100),
			Title: "Service", Place: 6,
			Lines: []model.Line{{Position: 1, Item: 5, Amount: model.AmountOf(100)}},
		},
		{
			Kind: model.KindExpense, SourceID: 301, VehicleSource: 11,
			OccurredAt: at(time.November, 12, 8), Odometer: 10700, Amount: amountOf(600),
			Lines: []model.Line{{Position: 1, Item: 7, Amount: model.AmountOf(600)}},
			Instalments: []model.Instalment{
				{Number: 1, DueOn: dayOn(2026, 11, 12), Amount: model.AmountOf(200)},
				{Number: 2, DueOn: dayOn(2026, 12, 12), Amount: model.AmountOf(200)},
				{Number: 3, DueOn: dayOn(2027, 1, 12), Amount: model.AmountOf(200)},
			},
		},
		{
			Kind: model.KindIncome, SourceID: 401, VehicleSource: 11,
			OccurredAt: at(time.November, 14, 8), Odometer: 10750, Amount: amountOf(50),
			IncomeType: 9,
		},
		{
			Kind: model.KindOdometer, SourceID: 501, VehicleSource: 11,
			OccurredAt: at(time.November, 20, 8), Odometer: 10900, Note: "Trip",
		},
	}
}

func hatchbackReminders() []model.Reminder {
	return []model.Reminder{
		{
			SourceID:      1,
			VehicleSource: 11,
			Kind:          model.KindExpense,
			Item:          7,
			DueOn:         new(dayOn(2026, 11, 30)),
		},
		{
			SourceID:      2,
			VehicleSource: 11,
			Kind:          model.KindMaintenance,
			Item:          5,
			DueOn:         new(dayOn(2026, 11, 20)),
		},
		{
			SourceID:      3,
			VehicleSource: 11,
			Kind:          model.KindExpense,
			Item:          7,
			DueOdometer:   new(11900.0),
		},
		{
			SourceID:      4,
			VehicleSource: 11,
			Kind:          model.KindMaintenance,
			Item:          5,
			DueOn:         new(dayOn(2026, 12, 25)),
		},
	}
}
