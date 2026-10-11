package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/problem"
)

const (
	pathCareIntervals = "/care-intervals"

	typeMisting = "misting"

	stateOnSchedule = "on_schedule"
	stateDue        = "due"
	stateOverdue    = "overdue"
)

type careAnswer struct {
	ID              string         `json:"id"`
	Plant           plantReference `json:"plant"`
	Type            string         `json:"type"`
	Days            int            `json:"days"`
	SetAt           string         `json:"set_at"`
	LastPerformedAt *string        `json:"last_performed_at"`
	CountedFrom     string         `json:"counted_from"`
	DaysSince       int            `json:"days_since"`
	State           *string        `json:"state"`
}

type carePage struct {
	Items []careAnswer `json:"items"`
	Next  string       `json:"next"`
}

func putInterval(
	t *testing.T,
	router http.Handler,
	plantID string,
	kind string,
	days int,
) *httptest.ResponseRecorder {
	t.Helper()

	return send(
		t,
		router,
		http.MethodPut,
		"/plants/"+plantID+pathCareIntervals+"/"+kind,
		fmt.Sprintf(`{"days":%d}`, days),
		"",
	)
}

// setInterval puts the care interval and requires that the hub took it.
func setInterval(
	t *testing.T,
	router http.Handler,
	plantID string,
	kind string,
	days int,
) careAnswer {
	t.Helper()

	recorder := putInterval(t, router, plantID, kind, days)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	return decode[careAnswer](t, recorder)
}

// careOf reads every care interval of the plant.
func careOf(t *testing.T, router http.Handler, plantID string) []careAnswer {
	t.Helper()

	recorder := send(t, router, http.MethodGet, "/plants/"+plantID+pathCareIntervals, "", "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	return decode[carePage](t, recorder).Items
}

// careOfType reads the one care interval of the type of the plant.
func careOfType(t *testing.T, router http.Handler, plantID string, kind string) careAnswer {
	t.Helper()

	for _, care := range careOf(t, router, plantID) {
		if care.Type == kind {
			return care
		}
	}

	require.Failf(t, "no care interval", "the plant holds no %s interval", kind)

	return careAnswer{}
}

// readEvery follows next from the link to the last page, and answers the items of
// every page.
func readEvery[T any](t *testing.T, router http.Handler, link string) []T {
	t.Helper()

	var items []T

	for link != "" {
		recorder := send(t, router, http.MethodGet, strings.TrimPrefix(link, externalURL), "", "")
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

		answered := decode[struct {
			Items []T    `json:"items"`
			Next  string `json:"next"`
		}](t, recorder)
		items = append(items, answered.Items...)
		link = answered.Next
	}

	return items
}

// moveSetAt moves the moment that the days of the care interval last changed to
// the count of days before now.
func moveSetAt(t *testing.T, fixture *garden, plantID string, kind string, days int) {
	t.Helper()

	require.NoError(t, fixture.inHome(t, func(tx *sqlx.Tx) error {
		if _, err := tx.Exec(
			`UPDATE care_intervals SET set_at = now() - make_interval(days => $3)
			 WHERE plant_id = $1 AND type = $2`,
			plantID, kind, days,
		); err != nil {
			return fmt.Errorf("moving the set moment: %w", err)
		}

		return nil
	}))
}

func daysAgo(days int) time.Time {
	return time.Now().UTC().AddDate(0, 0, -days)
}

func stateOf(care careAnswer) string {
	if care.State == nil {
		return ""
	}

	return *care.State
}

// GARDEN-SC-020, GARDEN-INV-003: An editor sets, changes, and removes care
// intervals. A plant holds one care interval for each type, and the days stay
// within 1 to 365.
func TestACareIntervalIsSetChangedAndRemoved(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")
	plant := makePlant(t, router, "Rose", environment.ID)

	setInterval(t, router, plant.ID, typeWatering, 7)
	setInterval(t, router, plant.ID, "fertilizing", 30)
	changed := setInterval(t, router, plant.ID, typeWatering, 10)
	assert.Equal(t, 10, changed.Days)

	for _, days := range []int{0, 366} {
		recorder := putInterval(t, router, plant.ID, "pruning", days)
		require.Equal(t, http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
		assert.Equal(t, []string{"#/days"}, pointersOf(t, recorder), "days %d", days)
	}

	removed := send(
		t,
		router,
		http.MethodDelete,
		"/plants/"+plant.ID+pathCareIntervals+"/fertilizing",
		"",
		"",
	)
	require.Equal(t, http.StatusNoContent, removed.Code, removed.Body.String())

	held := careOf(t, router, plant.ID)
	require.Len(t, held, 1)
	assert.Equal(t, typeWatering, held[0].Type)
	assert.Equal(t, 10, held[0].Days)
}

// GARDEN-FR-018: A path that names no maintenance type or no plant of the
// workspace names no care interval.
func TestACareIntervalOfNoTypeOrNoPlantIsNotFound(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)

	plant := makePlant(t, router, "Rose", makeEnvironment(t, router, "Office").ID)

	elsewhere := fixture.router(workspaceG)
	foreign := makePlant(t, elsewhere, "Rose", makeEnvironment(t, elsewhere, "Office").ID)

	for name, recorder := range map[string]*httptest.ResponseRecorder{
		"an unknown type":          putInterval(t, router, plant.ID, "repot", 7),
		"a plant of another space": putInterval(t, router, foreign.ID, typeWatering, 7),
		"a plant identifier that is no UUID": putInterval(
			t, router, "P001", typeWatering, 7,
		),
		"the intervals of a plant of another space": send(
			t, router, http.MethodGet, "/plants/"+foreign.ID+pathCareIntervals, "", "",
		),
	} {
		assert.Equal(t, http.StatusNotFound, recorder.Code, name+": "+recorder.Body.String())
	}
}

// GARDEN-SC-021: The days count in UTC calendar days from the date of the latest
// entry, whatever its time of day.
func TestTheDaysCountInCalendarDays(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	plant := makePlant(t, router, "Rose", makeEnvironment(t, router, "Office").ID)
	setInterval(t, router, plant.ID, typeWatering, 7)

	day := daysAgo(7)
	lateEvening := time.Date(day.Year(), day.Month(), day.Day(), 23, 30, 0, 0, time.UTC)
	recordEntry(t, router, typeWatering, lateEvening, plant.ID)

	care := careOfType(t, router, plant.ID, typeWatering)
	assert.Equal(t, 7, care.DaysSince)
	assert.Equal(t, lateEvening.Format(time.DateOnly), care.CountedFrom)
	require.NotNil(t, care.LastPerformedAt)
	assert.Equal(t, lateEvening, parseMoment(t, *care.LastPerformedAt))
}

// GARDEN-SC-022: With no entry, the days count from the moment the days last
// changed. A save with the same days keeps that moment.
func TestTheDaysCountFromTheIntervalWhenNoEntryExists(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)

	plant := makePlant(t, router, "Rose", makeEnvironment(t, router, "Office").ID)
	setInterval(t, router, plant.ID, typeWatering, 7)
	moveSetAt(t, fixture, plant.ID, typeWatering, 8)

	same := setInterval(t, router, plant.ID, typeWatering, 7)
	assert.Equal(t, 8, same.DaysSince)
	assert.Equal(t, stateDue, stateOf(same))
	assert.Nil(t, same.LastPerformedAt)

	longer := setInterval(t, router, plant.ID, typeWatering, 14)
	assert.Equal(t, 0, longer.DaysSince)
	assert.Equal(t, stateOnSchedule, stateOf(longer))
}

// GARDEN-SC-023: Watering every 7 days is on schedule at 6 days, due from 7 to 10
// days, and overdue at 11 days.
func TestTheStateFollowsTheDaysSinceTheLatestEntry(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")

	for days, state := range map[int]string{
		6:  stateOnSchedule,
		7:  stateDue,
		10: stateDue,
		11: stateOverdue,
	} {
		plant := makePlant(t, router, "Rose", environment.ID)
		setInterval(t, router, plant.ID, typeWatering, 7)
		recordEntry(t, router, typeWatering, daysAgo(days), plant.ID)

		care := careOfType(t, router, plant.ID, typeWatering)
		assert.Equal(t, days, care.DaysSince)
		assert.Equal(t, state, stateOf(care), "%d days", days)
	}
}

// GARDEN-SC-024: Misting every day is due at 1 day and overdue at 2. An inactive
// plant has no state, and the attention filter omits it.
func TestAnInactivePlantHasNoCareState(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	plant := makePlant(t, router, "Fern", makeEnvironment(t, router, "Office").ID)
	setInterval(t, router, plant.ID, typeMisting, 1)
	entry := recordEntry(t, router, typeMisting, daysAgo(1), plant.ID)[0]

	assert.Equal(t, stateDue, stateOf(careOfType(t, router, plant.ID, typeMisting)))

	moved := send(t, router, http.MethodPut, pathEntries+"/"+entry.ID, encode(t, map[string]any{
		memberType:        typeMisting,
		memberPerformedAt: daysAgo(2).Format(time.RFC3339),
	}), "")
	require.Equal(t, http.StatusOK, moved.Code, moved.Body.String())
	assert.Equal(t, stateOverdue, stateOf(careOfType(t, router, plant.ID, typeMisting)))

	inactive := replacePlant(t, router, plant, map[string]any{memberStatus: statusInactive})
	require.Equal(t, http.StatusOK, inactive.Code, inactive.Body.String())
	assert.Nil(t, careOfType(t, router, plant.ID, typeMisting).State)

	attention := readEvery[careAnswer](
		t,
		router,
		externalURL+pathCareIntervals+"?state=due,overdue",
	)
	assert.Empty(t, attention)
}

// GARDEN-SC-025: The environment list counts the active plants. The plant list
// keeps the active plants by default and filters by state and by environment. A
// cursor does not carry over to another filter.
func TestTheListsCountAndFilter(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	office := makeEnvironment(t, router, "Office")
	kitchen := makeEnvironment(t, router, "Kitchen")

	makePlant(t, router, "Rose", office.ID)
	makePlant(t, router, nameFern, office.ID)
	retired := makePlant(t, router, "Jasmine", office.ID)
	makePlant(t, router, "Basil", kitchen.ID)

	inactive := replacePlant(t, router, retired, map[string]any{memberStatus: statusInactive})
	require.Equal(t, http.StatusOK, inactive.Code, inactive.Body.String())

	environments := send(t, router, http.MethodGet, "/environments", "", "")
	require.Equal(t, http.StatusOK, environments.Code, environments.Body.String())

	counts := map[string]int{}
	for _, item := range decode[struct {
		Items []struct {
			DisplayID        string `json:"display_id"`
			ActivePlantCount int    `json:"active_plant_count"`
		} `json:"items"`
	}](t, environments).Items {
		counts[item.DisplayID] = item.ActivePlantCount
	}

	assert.Equal(t, map[string]int{"E001": 2, "E002": 1}, counts)

	for query, count := range map[string]int{
		"":                              3,
		"?status=inactive":              1,
		"?environment_id=" + kitchen.ID: 1,
	} {
		assert.Len(t, readEvery[plantAnswer](t, router, externalURL+"/plants"+query), count, query)
	}

	all := send(t, router, http.MethodGet, "/plants?limit=1", "", "")
	next := decode[carePage](t, all).Next
	require.NotEmpty(t, next)

	stale := send(
		t,
		router,
		http.MethodGet,
		strings.TrimPrefix(next, externalURL)+"&environment_id="+kitchen.ID,
		"",
		"",
	)
	require.Equal(t, http.StatusBadRequest, stale.Code, stale.Body.String())
	assert.Equal(t, problem.TypeCursorStale, problemTypeOf(t, stale))
}

// GARDEN-SC-026: A line of the attention list names the plant, its environment,
// the type, the days since the latest entry, the days of the interval, and the
// state.
func TestALineOfTheAttentionListNamesThePlantAndItsCare(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	office := makeEnvironment(t, router, "Office")
	makePlant(t, router, "Rose", office.ID)
	makePlant(t, router, nameFern, office.ID)
	gardenia := makePlant(t, router, "Gardenia", office.ID)

	setInterval(t, router, gardenia.ID, typeWatering, 7)
	recordEntry(t, router, typeWatering, daysAgo(9), gardenia.ID)

	attention := readEvery[careAnswer](
		t,
		router,
		externalURL+pathCareIntervals+"?state=due,overdue",
	)
	require.Len(t, attention, 1)

	line := attention[0]
	assert.Equal(t, "P003", line.Plant.DisplayID)
	assert.Equal(t, "Gardenia", line.Plant.CommonName)
	assert.Equal(t, "E001", line.Plant.Environment.DisplayID)
	assert.Equal(t, typeWatering, line.Type)
	assert.Equal(t, 9, line.DaysSince)
	assert.Equal(t, 7, line.Days)
	assert.Equal(t, stateDue, stateOf(line))
}

// GARDEN-FR-022: The care interval list filters by state and by environment, and
// refuses a filter value that names neither.
func TestTheCareIntervalListFilters(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	office := makeEnvironment(t, router, "Office")
	kitchen := makeEnvironment(t, router, "Kitchen")

	rose := makePlant(t, router, "Rose", office.ID)
	basil := makePlant(t, router, "Basil", kitchen.ID)

	setInterval(t, router, rose.ID, typeWatering, 7)
	setInterval(t, router, rose.ID, typeMisting, 2)
	setInterval(t, router, basil.ID, typeWatering, 7)
	recordEntry(t, router, typeMisting, daysAgo(2), rose.ID)

	for query, count := range map[string]int{
		"":                              3,
		"?state=on_schedule":            2,
		"?state=due":                    1,
		"?environment_id=" + kitchen.ID: 1,
		"?state=due&environment_id=" + kitchen.ID: 0,
	} {
		assert.Len(
			t,
			readEvery[careAnswer](t, router, externalURL+pathCareIntervals+query),
			count,
			query,
		)
	}

	for _, query := range []string{"state=late", "state=due,due", "environment_id=E001"} {
		recorder := send(t, router, http.MethodGet, pathCareIntervals+"?"+query, "", "")
		assert.Equal(t, http.StatusBadRequest, recorder.Code, query+": "+recorder.Body.String())
	}
}
