package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/drivvo/internal/fakes"
	"github.com/abgeo/maroid/plugins/drivvo/internal/fakeservice"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/widget"
)

const (
	unknownVehicle = "/vehicles/0199c5e2-8f3a-7c21-9b4e-000000000000"
	memberAmount   = "amount"
	memberCurrency = "currency"
)

type entryItem struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"`
	Odometer    float64        `json:"odometer"`
	Amount      map[string]any `json:"amount"`
	Labels      []string       `json:"labels"`
	Volume      *float64       `json:"volume"`
	UnitPrice   map[string]any `json:"unit_price"`
	FullTank    *bool          `json:"full_tank"`
	Consumption *float64       `json:"consumption"`
	Place       string         `json:"place"`
	IncomeType  string         `json:"income_type"`
}

func (r *pageRig) entries(t *testing.T, vehicle, query string) itemPage[entryItem] {
	t.Helper()

	return decodeAs[itemPage[entryItem]](t, r.get(t, r.vehiclePath(vehicle, "/log-entries")+query))
}

func (r *pageRig) entryOfOdometer(t *testing.T, odometer float64) entryItem {
	t.Helper()

	for _, item := range r.entries(t, hatchback, "?limit=100").Items {
		if item.Odometer == odometer {
			return item
		}
	}

	t.Fatalf("no entry at %v", odometer)

	return entryItem{}
}

// TestThePagesReadTheCopy runs each check of a read scenario of the pages, from
// DRIVVO-SC-029 to DRIVVO-SC-051, over one copy. Each check names its scenario.
func TestThePagesReadTheCopy(t *testing.T) {
	t.Parallel()

	rig := newPageRig(t, pagesSnapshot())

	for name, check := range map[string]func(*testing.T, *pageRig){
		"vehicles":                checkVehicles,
		"units":                   checkUnits,
		"vehicle not found":       checkVehicleNotFound,
		"period":                  checkPeriod,
		"overview":                checkOverview,
		"history":                 checkHistory,
		"lists":                   checkLists,
		"consumption of a refuel": checkConsumptionOfARefuel,
		"paging":                  checkPaging,
		"detail":                  checkDetail,
		"detail not found":        checkDetailNotFound,
		"reminders":               checkReminders,
		"widget":                  checkWidget,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			check(t, rig)
		})
	}
}

// DRIVVO-SC-033: Every read of a page answers from the copy, with no call to the service.
func TestThePagesCallNoService(t *testing.T) {
	t.Parallel()

	rig := newPageRig(t, pagesSnapshot())

	rig.fake.Set(func(state *fakeservice.Service) {
		state.Valid = map[string]bool{}
		state.SignInStatus = http.StatusInternalServerError
	})

	calls := len(rig.fake.Calls())

	for _, path := range readPaths(t, rig) {
		assert.Equal(t, http.StatusOK, rig.get(t, path).Code, path)
	}

	assert.Len(t, rig.fake.Calls(), calls)
}

// DRIVVO-SC-008: An end of the connection keeps the copy, so the history lists the same
// entries.
func TestTheCopyOutlivesTheConnection(t *testing.T) {
	t.Parallel()

	rig := newPageRig(t, pagesSnapshot())

	before := rig.entries(t, hatchback, "")

	assert.Equal(
		t,
		http.StatusNoContent,
		rig.serve(t, http.MethodDelete, "/connections/self").Code,
	)

	_, stored := rig.secrets.Session(fakes.WorkspaceW)
	assert.False(t, stored)
	assert.Equal(t, before, rig.entries(t, hatchback, ""))
}

// DRIVVO-SC-029: With no principal vehicle, the default is the first active one by name.
func TestTheDefaultWithNoPrincipalIsTheFirstActive(t *testing.T) {
	t.Parallel()

	rig := newPageRig(t, pagesSnapshot())

	snapshot := pagesSnapshot()
	for index := range snapshot.Vehicles {
		snapshot.Vehicles[index].Principal = false
	}

	rig.replace(t, snapshot)

	vehicles := decodeAs[itemPage[map[string]any]](t, rig.get(t, "/vehicles")).Items
	assert.Equal(t, hatchback, vehicles[0]["name"])
	assert.Equal(t, true, vehicles[0]["default"])
}

// DRIVVO-SC-029: The active vehicles come first by name, then the inactive one, and the
// principal vehicle is the default.
func checkVehicles(t *testing.T, rig *pageRig) {
	t.Helper()

	vehicles := decodeAs[itemPage[map[string]any]](t, rig.get(t, "/vehicles")).Items

	require.Len(t, vehicles, 3)
	assert.Equal(t, []any{hatchback, zeta, oldSedan},
		[]any{vehicles[0]["name"], vehicles[1]["name"], vehicles[2]["name"]})
	assert.Equal(t, []any{true, false, false},
		[]any{vehicles[0]["default"], vehicles[1]["default"], vehicles[2]["default"]})
}

// DRIVVO-SC-034: A vehicle names its units, and an amount travels as money in the
// currency of the settings.
func checkUnits(t *testing.T, rig *pageRig) {
	t.Helper()

	vehicles := decodeAs[itemPage[map[string]any]](t, rig.get(t, "/vehicles")).Items

	assert.Equal(t, map[string]any{
		"distance": "km", "volume": "l", "consumption_shape": "volume_per_100",
	}, vehicles[0]["units"])
	assert.Equal(t, map[string]any{memberAmount: 90.0, memberCurrency: currencyGEL},
		rig.entryOfOdometer(t, 10000).Amount)
}

// DRIVVO-SC-031: A vehicle that the copy does not hold answers 404 on each route.
func checkVehicleNotFound(t *testing.T, rig *pageRig) {
	t.Helper()

	for _, path := range []string{
		unknownVehicle + "/overview",
		unknownVehicle + "/log-entries",
		unknownVehicle + "/log-entry-totals?kind=refuel",
		unknownVehicle + "/reminders",
		"/vehicles/not-a-vehicle/overview",
	} {
		recorder := rig.get(t, path)
		assert.Equal(t, http.StatusNotFound, recorder.Code, path)
		assert.Equal(t, problemPrefix+"vehicle-not-found", problemTypeOf(t, recorder), path)
	}
}

// DRIVVO-SC-036: A moment counts on its day in the zone of the member, and a reversed
// period is refused.
func checkPeriod(t *testing.T, rig *pageRig) {
	t.Helper()

	overview := decodeAs[map[string]any](t,
		rig.get(t, rig.vehiclePath(oldSedan, "/overview?from=2026-11-01&to=2026-11-30")))

	assert.Equal(t, map[string]any{memberAmount: 120.0, memberCurrency: currencyGEL},
		objectOf(t, objectOf(t, overview, "spend"), "fuel"))

	reversed := rig.get(t, rig.vehiclePath(oldSedan, "/overview?from=2026-11-30&to=2026-11-01"))
	assert.Equal(t, http.StatusBadRequest, reversed.Code)
	assert.Equal(t, "/problems/http/request-invalid", problemTypeOf(t, reversed))
}

// DRIVVO-SC-038: The overview sums the spend of each kind, the instalments in the month
// of their due day, and the income apart.
func checkOverview(t *testing.T, rig *pageRig) {
	t.Helper()

	overview := decodeAs[map[string]any](t,
		rig.get(t, rig.vehiclePath(hatchback, "/overview?from=2026-11-01&to=2026-11-30")))

	spend := objectOf(t, overview, "spend")
	assert.InDelta(t, 240.0, objectOf(t, spend, "fuel")[memberAmount], 0.001)
	assert.InDelta(t, 100.0, objectOf(t, spend, "maintenance")[memberAmount], 0.001)
	assert.InDelta(t, 200.0, objectOf(t, spend, "expense")[memberAmount], 0.001)
	assert.InDelta(t, 540.0, objectOf(t, spend, "total")[memberAmount], 0.001)
	assert.InDelta(t, 50.0, objectOf(t, overview, "income")[memberAmount], 0.001)
	assert.Len(t, overview["months"], 1)
	assert.InDelta(t, 900.0, overview["distance"], 0.001)
	assert.InDelta(t, 6.25, objectOf(t, overview, "consumption")["value"], 0.0001)
}

// DRIVVO-SC-042: The history lists every entry, the latest first, and filters by kind.
func checkHistory(t *testing.T, rig *pageRig) {
	t.Helper()

	all := rig.entries(t, hatchback, "")

	require.Len(t, all.Items, 7)
	assert.Equal(t, "odometer", all.Items[0].Kind)
	assert.Nil(t, all.Items[0].Amount)

	chosen := rig.entries(t, hatchback, "?kind=refuel&kind=odometer")
	require.Len(t, chosen.Items, 4)

	for _, item := range chosen.Items {
		assert.Contains(t, []string{"refuel", "odometer"}, item.Kind)
	}

	assert.Equal(
		t,
		http.StatusBadRequest,
		rig.get(t, rig.vehiclePath(hatchback, "/log-entries?kind=trip")).Code,
	)
}

// DRIVVO-SC-043: Each list answers the members of its kind, and the totals sum the
// amount and the volume.
func checkLists(t *testing.T, rig *pageRig) {
	t.Helper()

	first := rig.entryOfOdometer(t, 10000)
	assert.Equal(t, []string{"Petrol", "LPG"}, first.Labels)
	assert.InDelta(t, 35.0, *first.Volume, 0.001)
	assert.Equal(t, map[string]any{memberAmount: 3.0, memberCurrency: currencyGEL}, first.UnitPrice)
	assert.True(t, *first.FullTank)

	assert.Equal(t, "Garage", rig.entryOfOdometer(t, 10600).Place)
	assert.Equal(t, "Ride", rig.entryOfOdometer(t, 10750).IncomeType)

	totals := decodeAs[map[string]any](t,
		rig.get(t, rig.vehiclePath(hatchback, "/log-entry-totals?kind=refuel")))
	assert.Equal(t, "refuel", totals["kind"])
	assert.InDelta(t, 3.0, totals["count"], 0)
	assert.InDelta(t, 240.0, objectOf(t, totals, memberAmount)[memberAmount], 0.001)
	assert.InDelta(t, 85.0, totals["volume"], 0.001)

	missing := rig.get(t, rig.vehiclePath(hatchback, "/log-entry-totals"))
	assert.Equal(t, http.StatusBadRequest, missing.Code)
}

// DRIVVO-SC-041: A refuel that ends a part with a value answers its consumption.
func checkConsumptionOfARefuel(t *testing.T, rig *pageRig) {
	t.Helper()

	assert.InDelta(t, 6.25, *rig.entryOfOdometer(t, 10800).Consumption, 0.0001)
	assert.Nil(t, rig.entryOfOdometer(t, 10450).Consumption)
}

// DRIVVO-SC-044: A list holds at most 20 entries, with links to the next and the
// previous ones.
func checkPaging(t *testing.T, rig *pageRig) {
	t.Helper()

	first := rig.entries(t, zeta, "")
	require.Len(t, first.Items, 20)
	assert.Nil(t, first.Prev)

	second := decodeAs[itemPage[entryItem]](t, rig.get(t, pathOf(t, first.Next)))
	require.Len(t, second.Items, 20)

	third := decodeAs[itemPage[entryItem]](t, rig.get(t, pathOf(t, second.Next)))
	require.Len(t, third.Items, 5)
	assert.Nil(t, third.Next)

	back := decodeAs[itemPage[entryItem]](t, rig.get(t, pathOf(t, third.Prev)))
	assert.Equal(t, second.Items, back.Items)
}

// DRIVVO-SC-045: The detail answers each value that the kind holds, and an odometer
// entry its odometer and its note alone.
func checkDetail(t *testing.T, rig *pageRig) {
	t.Helper()

	refuel := decodeAs[map[string]any](t, rig.get(t,
		rig.vehiclePath(hatchback, "/log-entries/"+rig.entryOfOdometer(t, 10000).ID)))

	assert.Len(t, refuel["lines"], 2)
	assert.Equal(t, "Gulf", refuel["station"])
	assert.Equal(t, "Card", refuel["payment_method"])
	assert.InDelta(t, 20.0, refuel["battery_start_pct"], 0.001)
	assert.InDelta(t, 80.0, refuel["battery_end_pct"], 0.001)
	assert.InDelta(t, 45.0, refuel["charge_minutes"], 0)

	odometer := decodeAs[map[string]any](t, rig.get(t,
		rig.vehiclePath(hatchback, "/log-entries/"+rig.entryOfOdometer(t, 10900).ID)))

	keys := make([]string, 0, len(odometer))
	for key := range odometer {
		keys = append(keys, key)
	}

	assert.ElementsMatch(t, []string{"id", "kind", "occurred_at", "odometer", "note"}, keys)
	assert.Equal(t, "Trip", odometer["note"])
}

// DRIVVO-SC-046: An entry of another vehicle, and an entry that the copy lacks, answer
// 404.
func checkDetailNotFound(t *testing.T, rig *pageRig) {
	t.Helper()

	other := rig.entries(t, oldSedan, "").Items[0].ID

	for _, path := range []string{
		rig.vehiclePath(hatchback, "/log-entries/"+other),
		rig.vehiclePath(hatchback, "/log-entries/0199c5e2-8f3a-7c21-9b4e-000000000000"),
		rig.vehiclePath(hatchback, "/log-entries/not-an-entry"),
	} {
		recorder := rig.get(t, path)
		assert.Equal(t, http.StatusNotFound, recorder.Code, path)
		assert.Equal(t, problemPrefix+"log-entry-not-found", problemTypeOf(t, recorder))
	}
}

// DRIVVO-SC-047, DRIVVO-SC-050: A one-time reminder keeps its own day, and the reminders
// come due first, then by the days left, then by the distance left.
func checkReminders(t *testing.T, rig *pageRig) {
	t.Helper()

	reminders := decodeAs[itemPage[map[string]any]](t,
		rig.get(t, rig.vehiclePath(hatchback, "/reminders"))).Items

	require.Len(t, reminders, 4)
	assert.Equal(t, true, reminders[0]["due"])
	assert.Equal(t, "Oil change", reminders[0]["type"])

	assert.Equal(t, "Insurance", reminders[1]["type"])
	assert.Equal(t, "2026-12-07", reminders[1]["due_on"])
	assert.InDelta(t, 12.0, reminders[1]["days_left"], 0)
	assert.Equal(t, false, reminders[1]["due"])

	assert.InDelta(t, 30.0, reminders[2]["days_left"], 0)
	assert.InDelta(t, 900.0, reminders[3]["distance_left"], 0.001)
	assert.NotContains(t, reminders[3], "due_on")
}

// DRIVVO-SC-051: The widget answers the latest refuel, the consumption, and the first
// reminder of the default vehicle.
func checkWidget(t *testing.T, rig *pageRig) {
	t.Helper()

	answer, err := widget.NewCar(rig.figures).Load(memberContext(t.Context()))
	require.NoError(t, err)
	require.NotNil(t, answer.List)
	require.Len(t, answer.List.Items, 3)

	assert.Equal(t, "2026-11-15 · 20.00 L · 60.00 GEL", answer.List.Items[0].Meta)
	assert.Equal(t, "6.3 L/100 km", answer.List.Items[1].Meta)
	assert.Equal(t, "Oil change", answer.List.Items[2].Label)
	assert.Equal(t, "due", answer.List.Items[2].Meta)
	assert.Equal(t, rig.vehiclePath(hatchback, "/overview"), answer.List.More)
}

// readPaths answers a read of every route under /vehicles.
func readPaths(t *testing.T, rig *pageRig) []string {
	t.Helper()

	entry := rig.entryOfOdometer(t, 10000).ID

	return []string{
		"/vehicles",
		rig.vehiclePath(hatchback, "/overview"),
		rig.vehiclePath(hatchback, "/log-entries"),
		rig.vehiclePath(hatchback, "/log-entry-totals?kind=refuel"),
		rig.vehiclePath(hatchback, "/log-entries/"+entry),
		rig.vehiclePath(hatchback, "/reminders"),
	}
}

// DRIVVO-SC-027: No answer of a connect or of a read of any route holds the token, the
// password, its MD5, or the name, the email address, the phone, or the licence of the holder.
func TestNoAnswerHoldsASecretOrTheHolder(t *testing.T) {
	t.Parallel()

	rig := newPageRig(t, pagesSnapshot())

	paths := readPaths(t, rig)
	bodies := make([]string, 0, len(paths)+2)
	bodies = append(bodies,
		rig.serve(t, http.MethodPut, "/connections/self").Body.String(),
		rig.get(t, "/connections/self").Body.String(),
	)

	for _, path := range paths {
		bodies = append(bodies, rig.get(t, path).Body.String())
	}

	for _, body := range bodies {
		for _, secret := range []string{
			fakeservice.Token, "secret", secretMD5(),
			fakeservice.HolderName, fakeservice.HolderEmail,
			fakeservice.HolderPhone, fakeservice.HolderLicence,
		} {
			assert.NotContains(t, body, secret)
		}
	}
}

// DRIVVO-SC-032: With no vehicle in the copy, the list is empty and the widget answers not
// set up with the link to the connection page.
func TestAnEmptyCopyListsNoVehicle(t *testing.T) {
	t.Parallel()

	rig := newPageRig(t, model.Snapshot{Settings: pagesSnapshot().Settings})

	assert.Empty(t, decodeAs[itemPage[map[string]any]](t, rig.get(t, "/vehicles")).Items)

	answer, err := widget.NewCar(rig.figures).Load(memberContext(t.Context()))
	require.NoError(t, err)
	assert.Equal(t, "not_set_up", string(answer.State))
	assert.Equal(t, "/account", answer.Link)
}

// objectOf answers the member of the object as an object.
func objectOf(t *testing.T, object map[string]any, member string) map[string]any {
	t.Helper()

	value, ok := object[member].(map[string]any)
	require.True(t, ok, "%s is not an object", member)

	return value
}
