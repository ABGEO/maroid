package handler_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pathEntries = "/maintenance-entries"

	memberPlantIDs    = "plant_ids"
	memberType        = "type"
	memberPerformedAt = "performed_at"
	memberNote        = "note"

	typeWatering = "watering"
)

type plantReference struct {
	ID          string    `json:"id"`
	DisplayID   string    `json:"display_id"`
	CommonName  string    `json:"common_name"`
	Environment reference `json:"environment"`
}

type entryAnswer struct {
	ID          string         `json:"id"`
	Plant       plantReference `json:"plant"`
	Type        string         `json:"type"`
	PerformedAt string         `json:"performed_at"`
	Note        *string        `json:"note"`
}

type entryList struct {
	Items []entryAnswer `json:"items"`
}

type entryPage struct {
	Items []entryAnswer `json:"items"`
	First string        `json:"first"`
	Next  string        `json:"next"`
	Prev  string        `json:"prev"`
}

func recordEntries(
	t *testing.T,
	router http.Handler,
	body map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()

	return send(t, router, http.MethodPost, pathEntries, encode(t, body), "")
}

// recordEntry records one entry of the type for each plant at the moment, and
// answers the created entries.
func recordEntry(
	t *testing.T,
	router http.Handler,
	kind string,
	moment time.Time,
	plantIDs ...string,
) []entryAnswer {
	t.Helper()

	recorder := recordEntries(t, router, map[string]any{
		memberPlantIDs:    plantIDs,
		memberType:        kind,
		memberPerformedAt: moment.Format(time.RFC3339),
	})
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())

	return decode[entryList](t, recorder).Items
}

func readEntryPage(t *testing.T, router http.Handler, link string) entryPage {
	t.Helper()

	recorder := send(t, router, http.MethodGet, strings.TrimPrefix(link, externalURL), "", "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	return decode[entryPage](t, recorder)
}

// readEveryEntry follows next from the link to the last page.
func readEveryEntry(t *testing.T, router http.Handler, link string) []string {
	t.Helper()

	var ids []string

	for link != "" {
		answered := readEntryPage(t, router, link)
		for _, item := range answered.Items {
			ids = append(ids, item.ID)
		}

		link = answered.Next
	}

	return ids
}

func parseMoment(t *testing.T, value string) time.Time {
	t.Helper()

	moment, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)

	return moment
}

// GARDEN-SC-012: Each of the eight maintenance types is accepted. A ninth value
// is refused.
func TestEveryMaintenanceTypeIsAccepted(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")
	plant := makePlant(t, router, nameFern, environment.ID)

	for _, kind := range []string{
		typeWatering, "fertilizing", "pruning", "repotting",
		"pest_treatment", "cleaning", "misting", "other",
	} {
		recorder := recordEntries(t, router, map[string]any{
			memberPlantIDs: []string{plant.ID},
			memberType:     kind,
		})
		require.Equal(t, http.StatusCreated, recorder.Code, kind+": "+recorder.Body.String())
		assert.Equal(t, kind, decode[entryList](t, recorder).Items[0].Type)
	}

	refused := recordEntries(t, router, map[string]any{
		memberPlantIDs: []string{plant.ID},
		memberType:     "repot",
	})
	require.Equal(t, http.StatusUnprocessableEntity, refused.Code, refused.Body.String())
	assert.Equal(t, []string{"#/type"}, pointersOf(t, refused))
}

// GARDEN-SC-013: An entry with no time takes the current time, and an entry with
// no note carries none.
func TestAnEntryWithNoTimeTakesTheCurrentTime(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")
	plant := makePlant(t, router, "Gardenia", environment.ID)

	// The wire carries whole seconds.
	before := time.Now().UTC().Truncate(time.Second)
	recorder := recordEntries(t, router, map[string]any{
		memberPlantIDs: []string{plant.ID},
		memberType:     typeWatering,
	})
	after := time.Now().UTC()

	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())

	items := decode[entryList](t, recorder).Items
	require.Len(t, items, 1)

	performedAt := parseMoment(t, items[0].PerformedAt)
	assert.False(t, performedAt.Before(before), "performed_at %s is before the call", performedAt)
	assert.False(t, performedAt.After(after), "performed_at %s is after the call", performedAt)
	assert.Nil(t, items[0].Note)
	assert.Equal(t, plant.DisplayID, items[0].Plant.DisplayID)
	assert.Equal(t, "E001", items[0].Plant.Environment.DisplayID)
}

// GARDEN-SC-014: An entry of yesterday is accepted. An entry in the future and a
// note of 1001 characters are refused.
func TestAnEntryInTheFutureIsRefused(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")
	plant := makePlant(t, router, "Gardenia", environment.ID)

	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	evening := time.Date(
		yesterday.Year(),
		yesterday.Month(),
		yesterday.Day(),
		19,
		0,
		0,
		0,
		time.UTC,
	)

	accepted := recordEntry(t, router, typeWatering, evening, plant.ID)
	assert.Equal(t, evening, parseMoment(t, accepted[0].PerformedAt))

	future := recordEntries(t, router, map[string]any{
		memberPlantIDs:    []string{plant.ID},
		memberType:        typeWatering,
		memberPerformedAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339),
	})
	require.Equal(t, http.StatusUnprocessableEntity, future.Code, future.Body.String())
	assert.Equal(t, []string{"#/performed_at"}, pointersOf(t, future))

	longNote := recordEntries(t, router, map[string]any{
		memberPlantIDs: []string{plant.ID},
		memberType:     typeWatering,
		memberNote:     strings.Repeat("a", 1001),
	})
	require.Equal(t, http.StatusUnprocessableEntity, longNote.Code, longNote.Body.String())
	assert.Equal(t, []string{"#/note"}, pointersOf(t, longNote))
}

// GARDEN-SC-015: One request records the same entry for 13 plants, in the order
// of the request.
func TestOneRequestRecordsAnEntryForEachPlant(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")

	plantIDs := make([]string, 0, 13)
	for range 13 {
		plantIDs = append(plantIDs, makePlant(t, router, nameFern, environment.ID).ID)
	}

	slices.Reverse(plantIDs)

	recorder := recordEntries(t, router, map[string]any{
		memberPlantIDs: plantIDs,
		memberType:     typeWatering,
		memberNote:     "Bottom watering",
	})
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())

	items := decode[entryList](t, recorder).Items
	require.Len(t, items, 13)

	for index, item := range items {
		assert.Equal(t, plantIDs[index], item.Plant.ID, "item %d", index)
		assert.Equal(t, items[0].PerformedAt, item.PerformedAt, "item %d", index)
		require.NotNil(t, item.Note)
		assert.Equal(t, "Bottom watering", *item.Note)
	}
}

// GARDEN-SC-016: A selection that names a plant of no record lands no entry. A
// selection above 100 plants, or with one plant two times, is refused.
func TestASelectionWithAnUnknownPlantLandsNoEntry(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)

	environment := makeEnvironment(t, router, "Office")

	plantIDs := make([]string, 0, 4)
	for range 3 {
		plantIDs = append(plantIDs, makePlant(t, router, nameFern, environment.ID).ID)
	}

	elsewhere := fixture.router(workspaceG)
	foreign := makePlant(t, elsewhere, nameFern, makeEnvironment(t, elsewhere, "Office").ID)

	unknown := recordEntries(t, router, map[string]any{
		memberPlantIDs: append(slices.Clone(plantIDs), foreign.ID),
		memberType:     typeWatering,
	})
	require.Equal(t, http.StatusUnprocessableEntity, unknown.Code, unknown.Body.String())
	assert.Equal(t, []string{"#/plant_ids/3"}, pointersOf(t, unknown))

	for _, plantID := range plantIDs {
		assert.Zero(t, countRows(t, fixture, "maintenance_entries", plantID))
	}

	tooMany := make([]string, 0, 101)
	for range 101 {
		tooMany = append(tooMany, plantIDs[0])
	}

	for name, selection := range map[string][]string{
		"101 plants":          tooMany,
		"one plant two times": {plantIDs[0], plantIDs[0]},
	} {
		recorder := recordEntries(t, router, map[string]any{
			memberPlantIDs: selection,
			memberType:     typeWatering,
		})
		require.Equal(
			t,
			http.StatusUnprocessableEntity,
			recorder.Code,
			name+": "+recorder.Body.String(),
		)
		assert.Equal(t, []string{"#/plant_ids"}, pointersOf(t, recorder), name)
	}
}

// GARDEN-SC-018: An editor changes the type, the time, and the note of an entry,
// then deletes it.
func TestAnEntryIsChangedAndDeleted(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")
	plant := makePlant(t, router, "Rose", environment.ID)
	entry := recordEntry(t, router, typeWatering, time.Now().UTC(), plant.ID)[0]

	nineDaysAgo := time.Now().UTC().AddDate(0, 0, -9).Truncate(time.Second)
	changed := send(t, router, http.MethodPut, pathEntries+"/"+entry.ID, encode(t, map[string]any{
		memberType:        "misting",
		memberPerformedAt: nineDaysAgo.Format(time.RFC3339),
		memberNote:        "Moved to the window",
	}), "")
	require.Equal(t, http.StatusOK, changed.Code, changed.Body.String())

	answer := decode[entryAnswer](t, changed)
	assert.Equal(t, "misting", answer.Type)
	assert.Equal(t, nineDaysAgo, parseMoment(t, answer.PerformedAt))
	require.NotNil(t, answer.Note)
	assert.Equal(t, "Moved to the window", *answer.Note)
	assert.Equal(t, plant.ID, answer.Plant.ID)

	deleted := send(t, router, http.MethodDelete, pathEntries+"/"+entry.ID, "", "")
	require.Equal(t, http.StatusNoContent, deleted.Code, deleted.Body.String())

	gone := send(t, router, http.MethodGet, pathEntries+"/"+entry.ID, "", "")
	assert.Equal(t, http.StatusNotFound, gone.Code, gone.Body.String())
}

// GARDEN-SC-019: An inactive plant takes an entry.
func TestAnInactivePlantTakesAnEntry(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")
	plant := makePlant(t, router, "Jasmine", environment.ID)

	inactive := replacePlant(t, router, plant, map[string]any{memberStatus: statusInactive})
	require.Equal(t, http.StatusOK, inactive.Code, inactive.Body.String())

	recorder := recordEntries(t, router, map[string]any{
		memberPlantIDs: []string{plant.ID},
		memberType:     "repotting",
		memberNote:     "merged with P008",
	})
	assert.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
}

// GARDEN-DD-008, GARDEN-FR-022: The log reads newest first by default, sorts a
// tie on the time by the identifier, reads the same pages back through prev, and
// sorts oldest first on request.
func TestTheLogReadsNewestFirstAcrossPages(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")
	rose := makePlant(t, router, "Rose", environment.ID)
	fern := makePlant(t, router, nameFern, environment.ID)

	now := time.Now().UTC().Truncate(time.Second)

	entries := make([]entryAnswer, 0, 5)
	entries = append(
		entries,
		recordEntry(t, router, typeWatering, now.Add(-time.Hour), rose.ID, fern.ID)...)
	entries = append(
		entries,
		recordEntry(t, router, typeWatering, now.Add(-3*time.Hour), rose.ID)...)
	entries = append(entries, recordEntry(t, router, "misting", now.Add(-2*time.Hour), fern.ID)...)
	entries = append(
		entries,
		recordEntry(t, router, typeWatering, now.Add(-30*time.Minute), rose.ID)...)

	slices.SortFunc(entries, func(a, b entryAnswer) int {
		if order := strings.Compare(b.PerformedAt, a.PerformedAt); order != 0 {
			return order
		}

		return strings.Compare(b.ID, a.ID)
	})

	newestFirst := make([]string, 0, len(entries))
	for _, entry := range entries {
		newestFirst = append(newestFirst, entry.ID)
	}

	assert.Equal(t, newestFirst, readEveryEntry(t, router, externalURL+pathEntries+"?limit=2"))

	oldestFirst := slices.Clone(newestFirst)
	slices.Reverse(oldestFirst)
	assert.Equal(
		t,
		oldestFirst,
		readEveryEntry(t, router, externalURL+pathEntries+"?limit=2&sort=performed_at"),
	)

	second := readEntryPage(
		t,
		router,
		readEntryPage(t, router, externalURL+pathEntries+"?limit=2").Next,
	)
	back := readEntryPage(t, router, readEntryPage(t, router, second.Next).Prev)
	assert.Equal(t, second.Items, back.Items)
}

// GARDEN-FR-022: The log keeps the entries of one plant or one type, and refuses a
// filter value that names neither.
func TestTheLogFiltersByPlantAndType(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")
	rose := makePlant(t, router, "Rose", environment.ID)
	fern := makePlant(t, router, nameFern, environment.ID)

	now := time.Now().UTC()
	recordEntry(t, router, typeWatering, now, rose.ID, fern.ID)
	recordEntry(t, router, "misting", now, fern.ID)

	assert.Len(t, readEveryEntry(t, router, externalURL+pathEntries+"?plant_id="+fern.ID), 2)
	assert.Len(t, readEveryEntry(t, router, externalURL+pathEntries+"?type=misting"), 1)
	assert.Empty(
		t,
		readEveryEntry(t, router, externalURL+pathEntries+"?plant_id="+rose.ID+"&type=misting"),
	)

	for _, query := range []string{"plant_id=P001", "type=repot", "sort=note"} {
		recorder := send(t, router, http.MethodGet, pathEntries+"?"+query, "", "")
		assert.Equal(t, http.StatusBadRequest, recorder.Code, query+": "+recorder.Body.String())
	}
}

// GARDEN-DD-008: A cursor whose boundary entry is gone answers an empty page that
// links to the first page.
func TestACursorWithAGoneBoundaryAnswersAnEmptyPage(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")
	plant := makePlant(t, router, "Rose", environment.ID)

	now := time.Now().UTC()
	for hours := range 3 {
		recordEntry(t, router, typeWatering, now.Add(-time.Duration(hours)*time.Hour), plant.ID)
	}

	first := readEntryPage(t, router, externalURL+pathEntries+"?limit=1")
	require.NotEmpty(t, first.Next)

	deleted := send(t, router, http.MethodDelete, pathEntries+"/"+first.Items[0].ID, "", "")
	require.Equal(t, http.StatusNoContent, deleted.Code, deleted.Body.String())

	stale := readEntryPage(t, router, first.Next)
	assert.Empty(t, stale.Items)
	assert.NotEmpty(t, stale.First)
}
