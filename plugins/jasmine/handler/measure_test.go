//go:build measure

package handler_test

import (
	"fmt"
	"net/http"
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
