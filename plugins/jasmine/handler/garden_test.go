package handler_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
)

const (
	typeEnvironmentOccupied = "/problems/dev.maroid.jasmine/environment-occupied"

	memberCommonName     = "common_name"
	memberEnvironmentID  = "environment_id"
	memberStatus         = "status"
	memberAcquiredOn     = "acquired_on"
	memberScientificName = "scientific_name"

	nameFern = "Fern"

	statusInactive = "inactive"
)

type reference struct {
	ID        string `json:"id"`
	DisplayID string `json:"display_id"`
	Name      string `json:"name"`
}

type plantAnswer struct {
	ID             string    `json:"id"`
	DisplayID      string    `json:"display_id"`
	CommonName     string    `json:"common_name"`
	ScientificName *string   `json:"scientific_name"`
	Environment    reference `json:"environment"`
	AcquiredOn     *string   `json:"acquired_on"`
	CareNotes      *string   `json:"care_notes"`
	Status         string    `json:"status"`
}

func decode[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	var answer T

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer), recorder.Body.String())

	return answer
}

func encode(t *testing.T, body map[string]any) string {
	t.Helper()

	raw, err := json.Marshal(body)
	require.NoError(t, err)

	return string(raw)
}

// pointersOf reads the pointer of each failed field of a validation problem.
func pointersOf(t *testing.T, recorder *httptest.ResponseRecorder) []string {
	t.Helper()

	answer := decode[problem.ValidationProblem](t, recorder)

	pointers := make([]string, 0, len(answer.Errors))
	for _, failure := range answer.Errors {
		pointers = append(pointers, failure.Pointer)
	}

	return pointers
}

func makeEnvironment(t *testing.T, router http.Handler, name string) reference {
	t.Helper()

	recorder := send(
		t,
		router,
		http.MethodPost,
		"/environments",
		encode(t, map[string]any{"name": name}),
		"",
	)
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())

	return decode[reference](t, recorder)
}

func makePlant(t *testing.T, router http.Handler, name string, environmentID string) plantAnswer {
	t.Helper()

	recorder := send(t, router, http.MethodPost, "/plants", encode(t, map[string]any{
		memberCommonName:    name,
		memberEnvironmentID: environmentID,
	}), "")
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())

	return decode[plantAnswer](t, recorder)
}

// replacePlant puts every value of the plant, with the overrides on top.
func replacePlant(
	t *testing.T,
	router http.Handler,
	plant plantAnswer,
	overrides map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()

	body := map[string]any{
		memberCommonName:    plant.CommonName,
		memberEnvironmentID: plant.Environment.ID,
		memberStatus:        plant.Status,
	}
	maps.Copy(body, overrides)

	return send(t, router, http.MethodPut, "/plants/"+plant.ID, encode(t, body), "")
}

// insertCare writes a care interval and a count of maintenance entries for the
// plant, in the acting workspace.
func insertCare(t *testing.T, fixture *garden, plantID string, intervals []string, entries int) {
	t.Helper()

	require.NoError(t, fixture.inHome(t, func(tx *sqlx.Tx) error {
		for _, kind := range intervals {
			if _, err := tx.Exec(
				`INSERT INTO care_intervals (plant_id, type, days) VALUES ($1, $2, 7)`,
				plantID,
				kind,
			); err != nil {
				return fmt.Errorf("inserting a care interval: %w", err)
			}
		}

		for range entries {
			if _, err := tx.Exec(
				`INSERT INTO maintenance_entries (plant_id, type, performed_at) VALUES ($1, 'watering', now())`,
				plantID,
			); err != nil {
				return fmt.Errorf("inserting a maintenance entry: %w", err)
			}
		}

		return nil
	}))
}

// countRows answers the rows of the table that name the plant, in H.
func countRows(t *testing.T, fixture *garden, table string, plantID string) int {
	t.Helper()

	var count int

	require.NoError(t, fixture.inHome(t, func(tx *sqlx.Tx) error {
		return tx.Get(&count, `SELECT count(*) FROM `+table+` WHERE plant_id = $1`, plantID)
	}))

	return count
}

// GARDEN-SC-001: An editor creates and renames an environment. A name of 101
// characters is refused.
func TestAnEnvironmentIsCreatedAndRenamed(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	created := makeEnvironment(t, router, "Office")
	assert.Equal(t, "Office", created.Name)

	renamed := send(t, router, http.MethodPut, "/environments/"+created.ID, `{"name":"Study"}`, "")
	require.Equal(t, http.StatusOK, renamed.Code, renamed.Body.String())
	assert.Equal(t, "Study", decode[reference](t, renamed).Name)

	tooLong := send(t, router, http.MethodPost, "/environments",
		encode(t, map[string]any{"name": strings.Repeat("a", 101)}), "")
	require.Equal(t, http.StatusUnprocessableEntity, tooLong.Code, tooLong.Body.String())
	assert.Equal(t, []string{"#/name"}, pointersOf(t, tooLong))
}

// GARDEN-SC-002: An environment that holds an inactive plant stays. It goes after
// the plant goes.
func TestAnEnvironmentThatHoldsAPlantIsNotDeleted(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")
	plant := makePlant(t, router, "Gardenia", environment.ID)

	inactive := replacePlant(t, router, plant, map[string]any{memberStatus: statusInactive})
	require.Equal(t, http.StatusOK, inactive.Code, inactive.Body.String())

	refused := send(t, router, http.MethodDelete, "/environments/"+environment.ID, "", "")
	require.Equal(t, http.StatusConflict, refused.Code, refused.Body.String())
	assert.Equal(t, typeEnvironmentOccupied, problemTypeOf(t, refused))

	kept := send(t, router, http.MethodGet, "/environments/"+environment.ID, "", "")
	assert.Equal(t, http.StatusOK, kept.Code)

	removedPlant := send(t, router, http.MethodDelete, "/plants/"+plant.ID, "", "")
	require.Equal(t, http.StatusNoContent, removedPlant.Code, removedPlant.Body.String())

	removed := send(t, router, http.MethodDelete, "/environments/"+environment.ID, "", "")
	assert.Equal(t, http.StatusNoContent, removed.Code, removed.Body.String())
}

// GARDEN-SC-003: A plant carries every value that the editor gives, and starts
// active.
func TestAPlantIsCreatedWithEveryValue(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")
	today := time.Now().UTC().Format(time.DateOnly)
	notes := strings.Repeat("a", 10000)

	recorder := send(t, router, http.MethodPost, "/plants", encode(t, map[string]any{
		memberCommonName:     "Gardenia",
		memberScientificName: "Gardenia jasminoides",
		memberEnvironmentID:  environment.ID,
		memberAcquiredOn:     today,
		"care_notes":         notes,
	}), "")
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
	assert.NotEmpty(t, recorder.Header().Get(precondition.ETagHeader))

	plant := decode[plantAnswer](t, recorder)
	assert.Equal(t, "Gardenia", plant.CommonName)
	require.NotNil(t, plant.ScientificName)
	assert.Equal(t, "Gardenia jasminoides", *plant.ScientificName)
	require.NotNil(t, plant.AcquiredOn)
	assert.Equal(t, today, *plant.AcquiredOn)
	require.NotNil(t, plant.CareNotes)
	assert.Equal(t, notes, *plant.CareNotes)
	assert.Equal(t, "active", plant.Status)
	assert.Equal(
		t,
		reference{ID: environment.ID, DisplayID: "E001", Name: "Office"},
		plant.Environment,
	)
}

// GARDEN-SC-004: A plant acquired tomorrow is accepted. A later date, a long
// scientific name, and an environment of another workspace are refused.
func TestAPlantRefusesAValueOutsideItsRules(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)

	environment := makeEnvironment(t, router, "Office")
	foreign := makeEnvironment(t, fixture.router(workspaceG), "Garage")
	tomorrow := time.Now().UTC().AddDate(0, 0, 1)

	accepted := send(t, router, http.MethodPost, "/plants", encode(t, map[string]any{
		memberCommonName:    nameFern,
		memberEnvironmentID: environment.ID,
		memberAcquiredOn:    tomorrow.Format(time.DateOnly),
	}), "")
	assert.Equal(t, http.StatusCreated, accepted.Code, accepted.Body.String())

	for pointer, body := range map[string]map[string]any{
		"#/acquired_on": {
			memberCommonName:    nameFern,
			memberEnvironmentID: environment.ID,
			memberAcquiredOn:    tomorrow.AddDate(0, 0, 1).Format(time.DateOnly),
		},
		"#/scientific_name": {
			memberCommonName:     nameFern,
			memberEnvironmentID:  environment.ID,
			memberScientificName: strings.Repeat("a", 151),
		},
		"#/environment_id": {
			memberCommonName:    nameFern,
			memberEnvironmentID: foreign.ID,
		},
	} {
		refused := send(t, router, http.MethodPost, "/plants", encode(t, body), "")
		require.Equal(t, http.StatusUnprocessableEntity, refused.Code, refused.Body.String())
		assert.Equal(t, []string{pointer}, pointersOf(t, refused))
	}
}

// GARDEN-SC-005: A plant that moves to another environment keeps its display
// identifier and its maintenance entries.
func TestAPlantThatMovesKeepsItsIdentifierAndHistory(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)

	office := makeEnvironment(t, router, "Office")
	kitchen := makeEnvironment(t, router, "Kitchen")
	plant := makePlant(t, router, "Gardenia", office.ID)
	recordEntry(t, router, typeWatering, time.Now().UTC(), plant.ID)
	recordEntry(t, router, "misting", time.Now().UTC(), plant.ID)

	moved := replacePlant(t, router, plant, map[string]any{
		memberEnvironmentID: kitchen.ID,
		memberCommonName:    "Cape jasmine",
	})
	require.Equal(t, http.StatusOK, moved.Code, moved.Body.String())

	answer := decode[plantAnswer](t, moved)
	assert.Equal(t, "P001", answer.DisplayID)
	assert.Equal(t, "E002", answer.Environment.DisplayID)
	assert.Equal(t, "Cape jasmine", answer.CommonName)
	assert.Len(t, readEveryEntry(t, router, externalURL+pathEntries+"?plant_id="+plant.ID), 2)
}

// GARDEN-SC-006: A new plant is active. The editor makes it inactive and active
// again. A state outside the two is refused.
func TestAPlantChangesItsStateInBothDirections(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")
	plant := makePlant(t, router, "Gardenia", environment.ID)
	require.Equal(t, "active", plant.Status)

	for _, status := range []string{statusInactive, "active"} {
		changed := replacePlant(t, router, plant, map[string]any{memberStatus: status})
		require.Equal(t, http.StatusOK, changed.Code, changed.Body.String())
		assert.Equal(t, status, decode[plantAnswer](t, changed).Status)
	}

	refused := replacePlant(t, router, plant, map[string]any{memberStatus: "dormant"})
	require.Equal(t, http.StatusUnprocessableEntity, refused.Code, refused.Body.String())
	assert.Equal(t, []string{"#/status"}, pointersOf(t, refused))
}

// GARDEN-SC-007: The delete of a plant removes its care intervals and its
// maintenance entries.
func TestTheDeleteOfAPlantRemovesItsCareAndHistory(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)

	environment := makeEnvironment(t, router, "Office")
	plant := makePlant(t, router, "Gardenia", environment.ID)
	insertCare(t, fixture, plant.ID, []string{"watering", "fertilizing"}, 3)

	removed := send(t, router, http.MethodDelete, "/plants/"+plant.ID, "", "")
	require.Equal(t, http.StatusNoContent, removed.Code, removed.Body.String())

	assert.Zero(t, countRows(t, fixture, "care_intervals", plant.ID))
	assert.Zero(t, countRows(t, fixture, "maintenance_entries", plant.ID))
}

// GARDEN-SC-009: Each workspace numbers its plants and its environments on its
// own, from 1.
func TestEachWorkspaceNumbersItsOwnRecords(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	home, other := fixture.router(workspaceH), fixture.router(workspaceG)

	office := makeEnvironment(t, home, "Office")
	for i := range 14 {
		makePlant(t, home, fmt.Sprintf("Plant %d", i+1), office.ID)
	}

	assert.Equal(t, "P015", makePlant(t, home, nameFern, office.ID).DisplayID)
	assert.Equal(t, "E002", makeEnvironment(t, home, "Kitchen").DisplayID)

	garage := makeEnvironment(t, other, "Garage")
	assert.Equal(t, "E001", garage.DisplayID)
	assert.Equal(t, "P001", makePlant(t, other, "Rose", garage.ID).DisplayID)
}

// GARDEN-SC-010: A deleted number never returns, and two creates at one moment
// never share a number.
func TestADisplayNumberIsNeverGivenTwice(t *testing.T) {
	t.Parallel()

	const writers = 20

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)

	office := makeEnvironment(t, router, "Office")

	var highest plantAnswer
	for i := range 15 {
		highest = makePlant(t, router, fmt.Sprintf("Plant %d", i+1), office.ID)
	}

	require.Equal(t, "P015", highest.DisplayID)

	removed := send(t, router, http.MethodDelete, "/plants/"+highest.ID, "", "")
	require.Equal(t, http.StatusNoContent, removed.Code)
	assert.Equal(t, "P016", makePlant(t, router, nameFern, office.ID).DisplayID)

	answers := make([]*httptest.ResponseRecorder, writers)

	var group sync.WaitGroup
	for i := range writers {
		group.Go(func() {
			answers[i] = send(t, router, http.MethodPost, "/plants", encode(t, map[string]any{
				memberCommonName:    fmt.Sprintf("Writer %d", i),
				memberEnvironmentID: office.ID,
			}), "")
		})
	}

	group.Wait()

	numbers := map[string]struct{}{}

	for _, answer := range answers {
		require.Equal(t, http.StatusCreated, answer.Code, answer.Body.String())
		numbers[decode[plantAnswer](t, answer).DisplayID] = struct{}{}
	}

	expected := map[string]struct{}{}
	for number := 17; number <= 16+writers; number++ {
		expected[fmt.Sprintf("P%03d", number)] = struct{}{}
	}

	assert.Equal(t, expected, numbers)

	duplicate := fixture.inHome(t, func(tx *sqlx.Tx) error {
		if _, err := tx.Exec(
			`INSERT INTO plants (number, common_name, environment_id) VALUES (16, 'Copy', $1)`,
			office.ID,
		); err != nil {
			return fmt.Errorf("inserting a second plant 16: %w", err)
		}

		return nil
	})
	require.Error(t, duplicate)
	assert.Contains(t, duplicate.Error(), "plants_workspace_number_key")
}

// GARDEN-SC-011: A write that names the display identifier is refused, and the
// plant keeps its values.
func TestADisplayIdentifierIsNotWritable(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	environment := makeEnvironment(t, router, "Office")
	plant := makePlant(t, router, "Gardenia", environment.ID)

	refused := replacePlant(
		t,
		router,
		plant,
		map[string]any{"display_id": "P099", memberCommonName: "Rose"},
	)
	require.Equal(t, http.StatusBadRequest, refused.Code, refused.Body.String())
	assert.Equal(t, problem.TypeMemberUnknown, problemTypeOf(t, refused))

	kept := send(t, router, http.MethodGet, "/plants/"+plant.ID, "", "")
	require.Equal(t, http.StatusOK, kept.Code)

	answer := decode[plantAnswer](t, kept)
	assert.Equal(t, "P001", answer.DisplayID)
	assert.Equal(t, "Gardenia", answer.CommonName)
}

// GARDEN-SC-030: The database refuses a plant that names the environment of
// another workspace, whatever code writes it.
func TestAPlantCannotNameTheEnvironmentOfAnotherWorkspace(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	foreign := makeEnvironment(t, fixture.router(workspaceG), "Garage")

	err := fixture.inHome(t, func(tx *sqlx.Tx) error {
		if _, execErr := tx.Exec(
			`INSERT INTO plants (number, common_name, environment_id) VALUES (1, 'Fern', $1)`,
			foreign.ID,
		); execErr != nil {
			return fmt.Errorf("inserting a plant of G into H: %w", execErr)
		}

		return nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plants_environment_fkey")
}

// GARDEN-INV-003: The database holds at most one care interval for each
// maintenance type of a plant. GARDEN-SC-020 drives the same rule through the
// route.
func TestAPlantHoldsOneCareIntervalForEachType(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)

	environment := makeEnvironment(t, router, "Office")
	plant := makePlant(t, router, "Gardenia", environment.ID)
	insertCare(t, fixture, plant.ID, []string{"watering"}, 0)

	err := fixture.inHome(t, func(tx *sqlx.Tx) error {
		if _, execErr := tx.Exec(
			`INSERT INTO care_intervals (plant_id, type, days) VALUES ($1, 'watering', 14)`,
			plant.ID,
		); execErr != nil {
			return fmt.Errorf("inserting a second watering interval: %w", execErr)
		}

		return nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "care_intervals_workspace_plant_type_key")
	assert.Equal(t, 1, countRows(t, fixture, "care_intervals", plant.ID))
}
