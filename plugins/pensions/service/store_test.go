package service_test

import (
	"context"
	"fmt"
	"io/fs"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/pensions/db"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

const (
	pluginID  = "dev.maroid.pensions"
	employerE = "204567890"

	sourcePerson   = "დასაქმებული"
	sourceEmployer = "დამსაქმებელი"
	sourceState    = "სახელმწიფო"
)

// hubFixtures stands in for the core migrations of the hub: the users that a scoped
// table references.
const hubFixtures = `
CREATE TABLE public.users (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.users (id) VALUES ('` + userA + `'), ('` + userB + `');`

// storeUnderTest starts a database, applies the migrations of the plugin, and returns
// the store, the handle that reads behind it, and the pool of the owner.
func storeUnderTest(t *testing.T) (*service.Store, pluginapi.PluginDB, *sqlx.DB) {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID(pluginID)
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	database := testdb.PluginDB(instance, identifier)

	return service.NewStore(database), database, instance.DB
}

// october15 is the day of every contribution and asset value of the store tests.
func october15() time.Time {
	return time.Date(2026, time.October, 15, 0, 0, 0, 0, time.UTC)
}

func contributionOf(basis uuid.UUID, source string, amount float64) model.Contribution {
	code := employerE

	return model.Contribution{
		BasisID:          basis,
		Source:           source,
		ContributionType: "ჩვეულებრივი",
		PaidOn:           october15(),
		SalaryYear:       new(2026),
		SalaryMonth:      new(9),
		GrossSalary:      5000,
		Amount:           new(amount),
		Units:            new(amount / 12.47),
		EmployerCode:     &code,
		Employer:         &model.Employer{Code: code, Name: "შპს მაგალითი"},
	}
}

func shareRow(portfolio int64, units float64) model.PortfolioShare {
	return model.PortfolioShare{
		PortfolioID: portfolio, PortfolioNameKa: "ზომიერი", PortfolioNameEn: "Balanced",
		AssetManagerNameKa: "სააგენტო", AssetManagerNameEn: "Agency", Ticker: "GPA-B",
		RiskTierNameKa: "საშუალო", RiskTierNameEn: "Medium", TargetShare: 100,
		CurrentShare: 98.7, Amount: units * 12.47, Units: units, UnitPrice: 12.47,
	}
}

func summaryOf(assetValue float64) model.AccountSummary {
	return model.AccountSummary{
		AssetValue: assetValue, Gain: 1904.10, PersonContributions: 3500,
		EmployerContributions: 3500, StateContributions: 3576.45, TotalContributions: 10576.45,
	}
}

func dailyPrices(portfolio int64, count int) []model.PortfolioUnitPrice {
	prices := make([]model.PortfolioUnitPrice, 0, count)
	first := time.Date(2019, time.January, 1, 0, 0, 0, 0, time.UTC)

	for i := range count {
		prices = append(prices, model.PortfolioUnitPrice{
			PortfolioID: portfolio,
			PricedOn:    first.AddDate(0, 0, i),
			UnitPrice:   1 + float64(i)/1000,
		})
	}

	return prices
}

func countIn(ctx context.Context, t *testing.T, database pluginapi.PluginDB, query string) int {
	t.Helper()

	var count int

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count, query)
	}))

	return count
}

// PENSION-SC-019: A second history that repeats the first 1964 prices and adds 21
// rewrites the 21 alone.
func TestTheUnitPricesRewriteOnlyWhatChanged(t *testing.T) {
	t.Parallel()

	_, database, _ := storeUnderTest(t)
	ctx := actingAs(t, userA)

	var written int64

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		_, err := upsertUnitPrices(ctx, tx, dailyPrices(7, 1964))

		return err
	}))
	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		written, err = upsertUnitPrices(ctx, tx, dailyPrices(7, 1985))

		return err
	}))

	assert.Equal(t, int64(21), written)
	assert.Equal(t, 1985, countIn(ctx, t, database,
		`SELECT count(*) FROM portfolio_unit_prices WHERE portfolio_id = 7;`))
}

// PENSION-SC-020: The summary is one row for each user, and the shares are those of the
// last snapshot alone.
func TestTheSummaryAndTheSharesHoldTheLastSnapshot(t *testing.T) {
	t.Parallel()

	store, database, _ := storeUnderTest(t)
	ctx := actingAs(t, userA)

	first := model.Snapshot{
		Summary: summaryOf(12000),
		Shares:  []model.PortfolioShare{shareRow(7, 600), shareRow(9, 400)},
	}
	first.Summary.Units = 1000
	require.NoError(t, store.Save(ctx, first))

	second := model.Snapshot{
		Summary: summaryOf(12480.55),
		Shares:  []model.PortfolioShare{shareRow(7, 1000.5)},
	}
	second.Summary.Units = 1000.5
	require.NoError(t, store.Save(ctx, second))

	var summaries []model.AccountSummary

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &summaries, `
			SELECT asset_value, units, gain, withheld_amount, person_contributions,
			       employer_contributions, state_contributions, total_contributions, collected_at
			FROM account_summaries;`)
	}))
	require.Len(t, summaries, 1)
	assert.InDelta(t, 12480.55, summaries[0].AssetValue, 1e-9)
	assert.InDelta(t, 1000.5, summaries[0].Units, 1e-9)

	var shares []struct {
		PortfolioID int64   `db:"portfolio_id"`
		Units       float64 `db:"units"`
	}

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &shares, `SELECT portfolio_id, units FROM portfolio_shares;`)
	}))
	require.Len(t, shares, 1)
	assert.Equal(t, int64(7), shares[0].PortfolioID)
	assert.InDelta(t, summaries[0].Units, shares[0].Units, 1e-9)
}

// PENSION-SC-022: A failure of the last statement of a save leaves every contribution as
// it was.
func TestAFailedSaveChangesNothing(t *testing.T) {
	t.Parallel()

	store, database, owner := storeUnderTest(t)
	ctx := actingAs(t, userA)

	held := []model.Contribution{
		contributionOf(uuid.New(), sourcePerson, 100),
		contributionOf(uuid.New(), sourcePerson, 110),
		contributionOf(uuid.New(), sourcePerson, 120),
	}
	require.NoError(t, store.Save(ctx, model.Snapshot{Contributions: held}))

	_, err := owner.Exec(`
		CREATE FUNCTION dev_maroid_pensions.refuse() RETURNS trigger LANGUAGE plpgsql AS
		$$ BEGIN RAISE EXCEPTION 'refused'; END $$;
		CREATE TRIGGER refuse BEFORE INSERT ON dev_maroid_pensions.portfolio_shares
		FOR EACH ROW EXECUTE FUNCTION dev_maroid_pensions.refuse();`)
	require.NoError(t, err)

	changed := make([]model.Contribution, 0, 5)

	for _, contribution := range held {
		contribution.Amount = new(999.0)
		changed = append(changed, contribution)
	}

	changed = append(changed,
		contributionOf(uuid.New(), sourcePerson, 130),
		contributionOf(uuid.New(), sourcePerson, 140),
	)

	err = store.Save(
		ctx,
		model.Snapshot{Contributions: changed, Shares: []model.PortfolioShare{shareRow(7, 1)}},
	)
	require.Error(t, err)

	assert.Equal(t, 3, countIn(ctx, t, database, `SELECT count(*) FROM contributions;`))
	assert.Equal(
		t,
		0,
		countIn(ctx, t, database, `SELECT count(*) FROM contributions WHERE amount = 999;`),
	)
}

// PENSION-SC-023: One basis holds one contribution of each source. A second save updates
// the amount of the employer, and adds no row.
func TestAContributionIsOneSourceOfOneBasis(t *testing.T) {
	t.Parallel()

	store, database, _ := storeUnderTest(t)
	ctx := actingAs(t, userA)
	basis := uuid.New()

	snapshot := func(employerAmount float64) model.Snapshot {
		return model.Snapshot{Contributions: []model.Contribution{
			contributionOf(basis, sourcePerson, 100),
			contributionOf(basis, sourceEmployer, employerAmount),
			contributionOf(basis, sourceState, 50),
		}}
	}

	require.NoError(t, store.Save(ctx, snapshot(100)))
	require.NoError(t, store.Save(ctx, snapshot(105)))

	assert.Equal(t, 3, countIn(ctx, t, database, `SELECT count(*) FROM contributions;`))
	assert.Equal(t, 1, countIn(ctx, t, database, fmt.Sprintf(
		`SELECT count(*) FROM contributions WHERE source = '%s' AND amount = 105;`,
		sourceEmployer,
	)))
	assert.Equal(t, 1, countIn(ctx, t, database, `SELECT count(*) FROM employers;`))
}

// PENSION-SC-024: One date holds one asset value. A second value of the date replaces it.
func TestADateHoldsOneAssetValue(t *testing.T) {
	t.Parallel()

	store, database, _ := storeUnderTest(t)
	ctx := actingAs(t, userA)

	value := model.AssetValue{
		ValuedOn:   october15(),
		AssetValue: 12000,
		Units:      1000,
		UnitPrice:  12,
	}
	require.NoError(t, store.Save(ctx, model.Snapshot{AssetValues: []model.AssetValue{value}}))

	value.AssetValue = 12010
	require.NoError(t, store.Save(ctx, model.Snapshot{AssetValues: []model.AssetValue{value}}))

	var values []model.AssetValue

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.SelectContext(
			ctx,
			&values,
			`SELECT valued_on, asset_value, units, unit_price FROM asset_values;`,
		)
	}))
	require.Len(t, values, 1)
	assert.Equal(t, october15(), values[0].ValuedOn.UTC())
	assert.InDelta(t, 12010.0, values[0].AssetValue, 1e-9)
}

// PENSION-SC-029: Another member of the same workspace reads no row of a user, and reads
// the shared employer and unit prices.
func TestAUserReadsNoRowOfAnotherUser(t *testing.T) {
	t.Parallel()

	store, database, _ := storeUnderTest(t)
	workspace := "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e70"
	ctxA := pluginapi.ContextWithActingWorkspace(actingAs(t, userA), workspace)
	ctxB := pluginapi.ContextWithActingWorkspace(actingAs(t, userB), workspace)

	summary := summaryOf(12480.55)
	require.NoError(t, store.Save(ctxA, model.Snapshot{
		Contributions: []model.Contribution{contributionOf(uuid.New(), sourcePerson, 100)},
		AssetValues: []model.AssetValue{
			{ValuedOn: october15(), AssetValue: 1, Units: 1, UnitPrice: 1},
		},
		UnitPrices: dailyPrices(7, 3),
		Summary:    summary,
		Shares:     []model.PortfolioShare{shareRow(7, 1)},
	}))

	for _, table := range []string{"contributions", "asset_values", "account_summaries", "portfolio_shares"} {
		assert.Zero(t, countIn(ctxB, t, database, `SELECT count(*) FROM `+table+`;`), table)
		assert.Equal(
			t,
			1,
			min(1, countIn(ctxA, t, database, `SELECT count(*) FROM `+table+`;`)),
			table,
		)
	}

	assert.Equal(t, 1, countIn(ctxB, t, database, `SELECT count(*) FROM employers;`))
	assert.Equal(t, 3, countIn(ctxB, t, database, `SELECT count(*) FROM portfolio_unit_prices;`))
}
