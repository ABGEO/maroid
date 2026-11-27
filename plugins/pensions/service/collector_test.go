package service_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/pensions/errs"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

func collectorOver(
	client service.APIClientService,
	secrets *memorySecrets,
) (*service.Collector, *recordingStore) {
	store := &recordingStore{}

	return service.NewCollector(client, storedCredentials{}, secrets, store), store
}

// PENSION-SC-015: A user with no session is not connected. The collection calls no
// service and writes nothing.
func TestTheCollectionOfAUserWithNoSessionCallsNothing(t *testing.T) {
	t.Parallel()

	fake, client := startService(t)
	collector, store := collectorOver(client, secretsHolding(""))

	_, err := collector.Collect(actingAs(t, userA), october)

	require.ErrorIs(t, err, errs.ErrNotConnected)
	assert.Zero(t, fake.callCount())
	assert.Empty(t, store.saved())
}

// PENSION-SC-016: The contributions of October ask for the first and the last day of
// October, and so does the chart of the asset value.
func TestTheCollectionAsksForTheDaysOfTheMonth(t *testing.T) {
	t.Parallel()

	fake, client := startService(t)
	collector, _ := collectorOver(client, secretsHolding(tokenOld))

	_, err := collector.Collect(actingAs(t, userA), october)
	require.NoError(t, err)

	contributions := fake.callsTo(pathContributions)
	require.Len(t, contributions, 1)
	assert.Equal(t, "2026-10-01", contributions[0].query["startDate"])
	assert.Equal(t, "2026-10-31", contributions[0].query["endDate"])
	assert.Equal(t, "100", contributions[0].query["pageSize"])
	assert.Equal(t, tokenOld, contributions[0].token)

	chart := fake.callsTo(pathChart)
	require.Len(t, chart, 1)
	assert.JSONEq(t, `{"dateFrom":"2026-10-01","dateTo":"2026-10-31"}`, chart[0].body)
}

// PENSION-SC-017: Every page of the month joins one snapshot. A month with no
// contribution is an empty list, and no error.
func TestTheCollectionReadsEveryPageOfTheContributions(t *testing.T) {
	t.Parallel()

	t.Run("three pages", func(t *testing.T) {
		t.Parallel()

		fake, client := startService(t)
		rows := contributionsOf(23)
		fake.on(http.MethodGet, pathContributions, answerPages(rows[:10], rows[10:20], rows[20:]))

		collector, store := collectorOver(client, secretsHolding(tokenOld))

		collection, err := collector.Collect(actingAs(t, userA), october)

		require.NoError(t, err)
		assert.Equal(t, 23, collection.Contributions)
		require.Len(t, store.saved(), 1)
		assert.Len(t, store.saved()[0].Contributions, 23)
		assert.Len(t, fake.callsTo(pathContributions), 3)
	})

	t.Run("no contribution", func(t *testing.T) {
		t.Parallel()

		fake, client := startService(t)
		fake.on(http.MethodGet, pathContributions, answerPages())

		collector, store := collectorOver(client, secretsHolding(tokenOld))

		collection, err := collector.Collect(actingAs(t, userA), october)

		require.NoError(t, err)
		assert.Zero(t, collection.Contributions)
		require.Len(t, store.saved(), 1)
		assert.Empty(t, store.saved()[0].Contributions)
	})
}

// PENSION-SC-018: Each date of the chart is one asset value with its value, its units,
// and its unit price. An empty chart is no asset value and no error.
func TestTheCollectionKeepsEachDateOfTheChart(t *testing.T) {
	t.Parallel()

	t.Run("two dates", func(t *testing.T) {
		t.Parallel()

		_, client := startService(t)
		collector, store := collectorOver(client, secretsHolding(tokenOld))

		_, err := collector.Collect(actingAs(t, userA), october)
		require.NoError(t, err)

		values := store.saved()[0].AssetValues
		require.Len(t, values, 2)
		assert.Equal(t, time.Date(2026, time.October, 15, 0, 0, 0, 0, time.UTC), values[0].ValuedOn)
		assert.InDelta(t, 12000.0, values[0].AssetValue, 1e-9)
		assert.InDelta(t, 1000.0, values[0].Units, 1e-9)
		assert.InDelta(t, 12.0, values[0].UnitPrice, 1e-9)
		assert.Equal(t, time.Date(2026, time.October, 31, 0, 0, 0, 0, time.UTC), values[1].ValuedOn)
	})

	t.Run("an empty chart", func(t *testing.T) {
		t.Parallel()

		fake, client := startService(t)
		fake.on(http.MethodPost, pathChart, answerJSON(map[string]any{
			"statusCode": 200, "errors": nil, keyResult: []any{},
		}))

		collector, store := collectorOver(client, secretsHolding(tokenOld))

		_, err := collector.Collect(actingAs(t, userA), october)

		require.NoError(t, err)
		assert.Empty(t, store.saved()[0].AssetValues)
	})
}

// PENSION-SC-018, PENSION-SC-019, PENSION-SC-020: The snapshot holds the summary of the
// participant with the units of the shares, each share, and the history of the unit
// price of each portfolio of the shares.
func TestTheSnapshotHoldsTheSummaryTheSharesAndTheUnitPrices(t *testing.T) {
	t.Parallel()

	fake, client := startService(t)
	fake.on(
		http.MethodGet,
		pathAssets,
		answerJSON([]map[string]any{shareOf(7, 600), shareOf(9, 400.5)}),
	)

	collector, store := collectorOver(client, secretsHolding(tokenOld))

	_, err := collector.Collect(actingAs(t, userA), october)
	require.NoError(t, err)

	snapshot := store.saved()[0]
	assert.InDelta(t, 12480.55, snapshot.Summary.AssetValue, 1e-9)
	assert.InDelta(t, 1904.10, snapshot.Summary.Gain, 1e-9)
	assert.InDelta(t, 10576.45, snapshot.Summary.TotalContributions, 1e-9)
	assert.InDelta(t, 1000.5, snapshot.Summary.Units, 1e-9)

	require.Len(t, snapshot.Shares, 2)
	assert.Equal(t, "Balanced", snapshot.Shares[0].PortfolioNameEn)
	assert.InDelta(t, 98.7, snapshot.Shares[0].CurrentShare, 1e-9)

	var asked struct {
		PortfolioIDs []int64 `json:"portfolioIds"` //nolint:tagliatelle // the service names it.
	}

	unitPrices := fake.callsTo(pathUnitPrices)
	require.Len(t, unitPrices, 1)
	require.NoError(t, json.Unmarshal([]byte(unitPrices[0].body), &asked))
	assert.Equal(t, []int64{7, 9}, asked.PortfolioIDs)

	require.Len(t, snapshot.UnitPrices, 2)
	assert.Equal(t, int64(7), snapshot.UnitPrices[0].PortfolioID)
	assert.Equal(
		t,
		time.Date(2026, time.October, 30, 0, 0, 0, 0, time.UTC),
		snapshot.UnitPrices[0].PricedOn,
	)
}

// PENSION-SC-021: A failure on page 2 of the contributions fails the collection with the
// status, and writes nothing.
func TestAFailedPageWritesNothing(t *testing.T) {
	t.Parallel()

	fake, client := startService(t)
	pages := answerPages(contributionsOf(1), contributionsOf(1), contributionsOf(1))

	fake.on(http.MethodGet, pathContributions, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pageNumber") == "2" {
			answerStatus(http.StatusInternalServerError)(w, r)

			return
		}

		pages(w, r)
	})

	collector, store := collectorOver(client, secretsHolding(tokenOld))

	_, err := collector.Collect(actingAs(t, userA), october)

	require.ErrorIs(t, err, errs.ErrRequestFailed)

	var status *service.StatusError
	require.ErrorAs(t, err, &status)
	assert.Equal(t, http.StatusInternalServerError, status.Status)
	assert.Empty(t, store.saved())
}

// PENSION-SC-010: A rejected session gives way to one new sign in. The collection
// continues with the new session, and stores it.
func TestTheCollectionSignsInAgainOnce(t *testing.T) {
	t.Parallel()

	fake, client := startService(t)
	participant := answerJSON(participantOf("Nino", "Beridze", 12480.55))

	fake.on(http.MethodGet, pathParticipant, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer "+tokenOld {
			answerStatus(http.StatusUnauthorized)(w, r)

			return
		}

		participant(w, r)
	})

	secrets := secretsHolding(tokenOld)
	collector, store := collectorOver(client, secrets)

	_, err := collector.Collect(actingAs(t, userA), october)

	require.NoError(t, err)
	assert.Len(t, fake.callsTo(pathSignIn), 1)
	assert.Len(t, store.saved(), 1)

	token, found := secrets.token(t)
	require.True(t, found)
	assert.Equal(t, tokenNew, token)

	for _, recorded := range fake.callsTo(pathContributions) {
		assert.Equal(t, tokenNew, recorded.token)
	}
}

// PENSION-SC-012: Credentials that the sign in rejects remove the session, and fail the
// collection.
func TestTheCollectionFailsWhenTheCredentialsAreRejected(t *testing.T) {
	t.Parallel()

	fake, client := startService(t)
	fake.on(http.MethodGet, pathParticipant, answerStatus(http.StatusUnauthorized))
	fake.on(http.MethodPost, pathSignIn, answerStatus(http.StatusBadRequest))

	secrets := secretsHolding(tokenOld)
	collector, store := collectorOver(client, secrets)

	_, err := collector.Collect(actingAs(t, userA), october)

	require.ErrorIs(t, err, errs.ErrCredentialsRejected)
	assert.Empty(t, store.saved())

	_, found := secrets.token(t)
	assert.False(t, found)
}
