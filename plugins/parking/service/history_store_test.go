package service_test

import (
	"context"
	"io/fs"
	"log/slog"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/parking/db"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/model"
	"github.com/abgeo/maroid/plugins/parking/service"
)

const pluginID = "dev.maroid.parking"

// hubFixtures stands in for the core migrations of the hub: the users that a scoped
// table references.
const hubFixtures = `
CREATE TABLE public.users (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.users (id) VALUES ('` + userA + `'), ('` + userB + `');`

// storeUnderTest starts a database, applies the migrations of the plugin, and returns
// the store and the handle that reads behind it.
func storeUnderTest(t *testing.T) (*service.HistoryStore, pluginapi.PluginDB) {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID(pluginID)
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	database := testdb.PluginDB(instance, identifier)

	return service.NewHistoryStore(database), database
}

func countIn(ctx context.Context, t *testing.T, database pluginapi.PluginDB, query string) int {
	t.Helper()

	var count int

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count, query)
	}))

	return count
}

func sessionModel(id int64, amount float64) model.Session {
	started := time.Date(2026, time.November, 21, 9, 26, 8, 0, time.UTC)

	return model.Session{
		ExternalID: id, LotNumber: "A1637", Address: "12 Example Street", Plate: "AB071EO",
		ParkingType: "BOTH", StartedAt: started, EndedAt: started.Add(48 * time.Minute),
		Amount: amount,
	}
}

func transactionModel(id int64, session *int64) model.Transaction {
	return model.Transaction{
		ExternalID: id, Kind: model.TransactionKindParkingPayment,
		ServiceKind: "ZONE_PARKING_PAYMENT", Amount: 0.55, BalanceAfter: 9.08,
		OccurredAt:        time.Date(2026, time.November, 21, 10, 14, 21, 0, time.UTC),
		SessionExternalID: session, Description: "Zone Parking",
	}
}

func tenSessions() []model.Session {
	sessions := make([]model.Session, 10)
	for i := range sessions {
		sessions[i] = sessionModel(int64(100+i), 0.55)
	}

	return sessions
}

// PARKHIST-SC-011: A second save of a session keeps one row. An equal save changes no
// collected_at, and a corrected amount replaces the held amount.
func TestASessionIsOneRowOfItsIdentifier(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := as(userA)

	first, err := store.Save(ctx, []model.Session{sessionModel(45304769, 0.55)}, nil)
	require.NoError(t, err)
	assert.Equal(t, model.Collection{Sessions: 1}, first)

	read := func() (float64, time.Time) {
		var row struct {
			Amount    float64   `db:"amount"`
			Collected time.Time `db:"collected_at"`
		}

		require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
			return tx.GetContext(ctx, &row, `SELECT amount, collected_at FROM sessions;`)
		}))

		return row.Amount, row.Collected
	}

	_, collected := read()

	again, err := store.Save(ctx, []model.Session{sessionModel(45304769, 0.55)}, nil)
	require.NoError(t, err)
	assert.Equal(t, model.Collection{}, again)

	_, unchanged := read()
	assert.Equal(t, collected, unchanged)

	corrected, err := store.Save(ctx, []model.Session{sessionModel(45304769, 0.60)}, nil)
	require.NoError(t, err)
	assert.Equal(t, model.Collection{}, corrected)

	amount, _ := read()
	assert.InDelta(t, 0.60, amount, 1e-9)
	assert.Equal(t, 1, countIn(ctx, t, database, `SELECT count(*) FROM sessions;`))
}

// PARKHIST-SC-016: Another member of the same workspace reads no row of a user. A write
// with the external identifier of a row of another user adds a row of the writer.
func TestAUserReadsNoHistoryOfAnotherUser(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	workspace := "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e70"
	ctxA := pluginapi.ContextWithActingWorkspace(as(userA), workspace)
	ctxB := pluginapi.ContextWithActingWorkspace(as(userB), workspace)

	_, err := store.Save(ctxB,
		[]model.Session{sessionModel(7, 1.10)},
		[]model.Transaction{transactionModel(70, new(int64(7)))},
	)
	require.NoError(t, err)

	for _, table := range []string{"sessions", listTransactions} {
		assert.Zero(t, countIn(ctxA, t, database, `SELECT count(*) FROM `+table+`;`), table)
	}

	held, err := store.KnownSessions(ctxA, []int64{7})
	require.NoError(t, err)
	assert.Empty(t, held)

	collection, err := store.Save(ctxA,
		[]model.Session{sessionModel(7, 0.20)},
		[]model.Transaction{transactionModel(70, nil)},
	)
	require.NoError(t, err)
	assert.Equal(t, model.Collection{Sessions: 1, Transactions: 1}, collection)

	var amountOfB float64

	require.NoError(t, database.WithTx(ctxB, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctxB, &amountOfB, `SELECT amount FROM sessions;`)
	}))
	assert.InDelta(t, 1.10, amountOfB, 1e-9)
}

// PARKHIST-SC-008: A failed page of the transactions fails the full reach, and leaves the
// rows of the user as they were.
func TestAFailedCollectionLeavesTheRowsAsTheyWere(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := as(userA)

	_, err := store.Save(ctx, tenSessions(), nil)
	require.NoError(t, err)

	parking := &fakeParking{
		sessions:     sessionsOf(30),
		transactions: entriesOf(60),
		failAt:       map[string]int{listTransactions: 2},
	}
	collector := service.NewCollector(parking, store, slog.New(slog.DiscardHandler))

	_, err = collector.Collect(ctx, model.ReachFull())

	require.ErrorIs(t, err, errs.ErrRequestFailed)
	assert.Equal(t, 10, countIn(ctx, t, database, `SELECT count(*) FROM sessions;`))
	assert.Zero(t, countIn(ctx, t, database, `SELECT count(*) FROM transactions;`))
}

// PARKHIST-SC-008: A failed statement of a save rolls back the sessions of the save.
func TestAFailedSaveWritesNoSession(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := as(userA)

	bad := transactionModel(1, nil)
	bad.Kind = "refund"

	_, err := store.Save(ctx, tenSessions(), []model.Transaction{bad})

	require.Error(t, err)
	assert.Zero(t, countIn(ctx, t, database, `SELECT count(*) FROM sessions;`))
}
