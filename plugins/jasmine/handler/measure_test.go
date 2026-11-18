//go:build measure

package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// measuredGarden holds 500 active plants in 10 environments, one care interval of
// each of the 8 types for each plant, and 200 entries for each plant spread over
// 365 days.
const measuredGarden = `
INSERT INTO environments (number, name)
SELECT n, 'Room ' || n FROM generate_series(1, 10) AS n;

INSERT INTO plants (number, common_name, environment_id)
SELECT n, 'Plant ' || n, e.id
FROM generate_series(1, 500) AS n
JOIN environments e ON e.number = 1 + n % 10;

INSERT INTO care_intervals (plant_id, type, days)
SELECT p.id, t.type, 1 + (p.number + t.position) % 30
FROM plants p
CROSS JOIN unnest(ARRAY['watering', 'fertilizing', 'pruning', 'repotting',
                        'pest_treatment', 'cleaning', 'misting', 'other'])
     WITH ORDINALITY AS t (type, position);

INSERT INTO maintenance_entries (plant_id, type, performed_at)
SELECT p.id,
       (ARRAY['watering', 'fertilizing', 'pruning', 'repotting',
              'pest_treatment', 'cleaning', 'misting', 'other'])[1 + (p.number + s) % 8],
       now() - make_interval(secs => (p.number * 7919 + s * 104729) % 31536000)
FROM plants p
CROSS JOIN generate_series(1, 200) AS s;

ANALYZE;`

// GARDEN-SC-032: The attention list of a workspace with 500 plants, 4000 care
// intervals, and 100000 entries answers in less than 1 second, from the call of
// the handler to the last byte.
//
//nolint:paralleltest // a measurement runs alone, so no other test takes its time
func TestTheAttentionListAnswersWithinOneSecond(t *testing.T) {
	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)

	require.NoError(t, fixture.inHome(t, func(tx *sqlx.Tx) error {
		if _, err := tx.Exec(measuredGarden); err != nil {
			return fmt.Errorf("seeding the measured garden: %w", err)
		}

		return nil
	}))

	var slowest time.Duration

	for range 20 {
		start := time.Now()
		recorder := send(
			t,
			router,
			http.MethodGet,
			"/care-intervals?state=due,overdue&limit=100",
			"",
			"",
		)
		elapsed := time.Since(start)

		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		require.Len(t, decode[carePage](t, recorder).Items, 100, "the read answers a full page")

		slowest = max(slowest, elapsed)
	}

	t.Logf("the slowest of 20 reads took %s", slowest)
	assert.Less(t, slowest, time.Second)
}

// GARDEN-SC-046: The upcoming list of a workspace with 500 plants, 4000 care
// intervals, and 100000 entries answers in less than 1 second, from the call of
// the handler to the last byte.
//
//nolint:paralleltest // a measurement runs alone, so no other test takes its time
func TestTheUpcomingListAnswersWithinOneSecond(t *testing.T) {
	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)

	require.NoError(t, fixture.inHome(t, func(tx *sqlx.Tx) error {
		if _, err := tx.Exec(measuredGarden); err != nil {
			return fmt.Errorf("seeding the measured garden: %w", err)
		}

		return nil
	}))

	var slowest time.Duration

	for range 20 {
		start := time.Now()
		recorder := send(
			t,
			router,
			http.MethodGet,
			"/care-intervals?due_within=3&limit=100",
			"",
			"",
		)
		elapsed := time.Since(start)

		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		require.NotEmpty(t, decode[carePage](t, recorder).Items, "the read answers upcoming care")

		slowest = max(slowest, elapsed)
	}

	t.Logf("the slowest of 20 reads took %s", slowest)
	assert.Less(t, slowest, time.Second)
}

type plantPage struct {
	Items []plantAnswer `json:"items"`
	Next  string        `json:"next"`
}

// seedPhotos gives each moment the count of photos, each of a new ready file of H,
// and makes a photo of each plant its profile picture. The parent and the child rows
// land in one statement, where the keys hold at its end.
func seedPhotos(perMoment int) string {
	return fmt.Sprintf(`
WITH seed AS (
    SELECT m.id AS moment_id, m.plant_id, uuidv7() AS file_id
    FROM moments m
    CROSS JOIN generate_series(1, %d)
),
files AS (
    INSERT INTO public.workspace_files (id, workspace_id)
    SELECT file_id, '%s' FROM seed
)
INSERT INTO photos (plant_id, moment_id, file_id)
SELECT plant_id, moment_id, file_id FROM seed;

UPDATE plants p
SET profile_photo_id = (SELECT min(ph.id::text)::uuid FROM photos ph WHERE ph.plant_id = p.id)
WHERE EXISTS (SELECT 1 FROM photos ph WHERE ph.plant_id = p.id);

ANALYZE;`, perMoment, workspaceH)
}

// measuredGallery gives the plant number 1 of measuredGarden 100 moments.
const measuredGallery = `
INSERT INTO moments (plant_id, taken_on)
SELECT p.id, current_date - n
FROM plants p
CROSS JOIN generate_series(1, 100) AS n
WHERE p.number = 1;`

// measuredMoments gives each plant of measuredGarden one moment.
const measuredMoments = `
INSERT INTO moments (plant_id, taken_on) SELECT id, current_date FROM plants;`

// seed runs the statements in H.
func seed(t *testing.T, fixture *garden, statements ...string) {
	t.Helper()

	require.NoError(t, fixture.inHome(t, func(tx *sqlx.Tx) error {
		for _, statement := range statements {
			if _, err := tx.Exec(statement); err != nil {
				return fmt.Errorf("seeding the measured garden: %w", err)
			}
		}

		return nil
	}))
}

// GARDEN-SC-069: The gallery of a plant with 100 moments and 300 photos answers in
// less than 1 second, from the call of the handler to the last byte.
//
//nolint:paralleltest // a measurement runs alone, so no other test takes its time
func TestTheGalleryAnswersWithinOneSecond(t *testing.T) {
	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)

	seed(t, fixture, measuredGarden, measuredGallery, seedPhotos(3))

	var plantID string

	require.NoError(t, fixture.inHome(t, func(tx *sqlx.Tx) error {
		return tx.Get(&plantID, `SELECT id FROM plants WHERE number = 1`)
	}))

	var slowest time.Duration

	for range 20 {
		start := time.Now()
		recorder := send(t, router, http.MethodGet, momentsOf(plantID)+"?limit=100", "", "")
		elapsed := time.Since(start)

		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

		moments := decode[momentPage](t, recorder).Items
		require.Len(t, moments, 100, "the read answers every moment")
		require.NotNil(t, moments[99].Photos[2].Thumbnail, "the read answers the addresses")

		slowest = max(slowest, elapsed)
	}

	t.Logf("the slowest of 20 reads took %s", slowest)
	assert.Less(t, slowest, time.Second)
}

// GARDEN-SC-070: Each page of the plants of a workspace with 500 plants, each with a
// profile picture, answers in less than 1 second, from the call of the handler to the
// last byte.
//
//nolint:paralleltest // a measurement runs alone, so no other test takes its time
func TestThePlantPagesAnswerWithinOneSecond(t *testing.T) {
	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)

	seed(t, fixture, measuredGarden, measuredMoments, seedPhotos(1))

	var (
		slowest time.Duration
		plants  int
	)

	for link := "/plants?limit=100"; link != ""; {
		start := time.Now()
		recorder := send(t, router, http.MethodGet, link, "", "")
		elapsed := time.Since(start)

		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

		answered := decode[plantPage](t, recorder)
		for _, plant := range answered.Items {
			require.NotNil(t, plant.ProfilePhoto, "every plant answers its profile picture")
			require.NotNil(t, plant.ProfilePhoto.Thumbnail, "every picture answers its address")
		}

		plants += len(answered.Items)
		slowest = max(slowest, elapsed)
		link = strings.TrimPrefix(answered.Next, externalURL)
	}

	require.Equal(t, 500, plants, "the pages answer every plant")
	t.Logf("the slowest of %d pages took %s", plants/100, slowest)
	assert.Less(t, slowest, time.Second)
}
