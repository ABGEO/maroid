package service_test

import (
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/db"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/errs"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/service"
)

const (
	pluginID   = "dev.maroid.tbilisi-energy"
	workspaceW = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"
	workspaceX = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e60"
	customerC1 = "757235-992"
	customerC2 = "340309-649"

	operationReading = 125
	operationControl = 172
	operationPayment = 196
)

// hubFixtures stands in for the core migrations of the hub: the workspaces that a
// scoped table references.
const hubFixtures = `
CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.workspaces (id) VALUES ('` + workspaceW + `'), ('` + workspaceX + `');`

// storeUnderTest starts a database, applies the migrations of the plugin, and
// returns the store and the handle that reads behind it.
func storeUnderTest(t *testing.T) (*service.Store, *pluginapi.PluginDB) {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID(pluginID)
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	database := pluginapi.NewPluginDB(instance.DB, identifier)

	return service.NewStore(database), database
}

func actingIn(t *testing.T, workspaceID string) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingWorkspace(t.Context(), workspaceID)
}

func dateOf(month time.Month, day int) time.Time {
	return time.Date(2026, month, day, 0, 0, 0, 0, model.Tbilisi())
}

func readingOf(customer string, date time.Time, meterValue, amount float64) model.Transaction {
	bill := "api/Customer/GetCustomerBillPdf/" + date.Format(time.DateOnly) + ".pdf"

	return model.Transaction{
		CustomerNumber: customer,
		TypeID:         operationReading,
		Operation:      "რეალური ჩვენება",
		Date:           date,
		Consumption:    21,
		MeterValue:     meterValue,
		Amount:         amount,
		Balance:        amount,
		BillPath:       &bill,
	}
}

// paymentOf builds a payment of customer C1.
func paymentOf(date time.Time, amount float64) model.Transaction {
	return model.Transaction{
		CustomerNumber: customerC1,
		TypeID:         operationPayment,
		Operation:      "გაზის საფასურის ელ-გადახდა",
		Date:           date,
		Amount:         amount,
	}
}

func countOf(ctx context.Context, t *testing.T, database *pluginapi.PluginDB, table string) int {
	t.Helper()

	var count int

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count, `SELECT count(*) FROM `+table+`;`)
	}))

	return count
}

func datesOf(transactions []model.Transaction) []string {
	dates := make([]string, 0, len(transactions))
	for _, transaction := range transactions {
		dates = append(dates, transaction.Date.Format(time.DateOnly)+"/"+transaction.Operation)
	}

	return dates
}

// GASACC-SC-017: A reading and a payment of one date are two transactions. A repeat
// stores nothing, and a corrected amount replaces the amount of its date. Only the first
// insert of a row counts as stored for the first time.
func TestTheStoreKeepsOneTransactionOfAnOperationOnADate(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	reading := readingOf(customerC1, dateOf(time.September, 28), 1532, 11.33)
	payment := paymentOf(dateOf(time.September, 28), 11.33)
	corrected := readingOf(customerC1, dateOf(time.September, 28), 1532, 11.50)

	first, err := store.InsertTransactions(ctx, []model.Transaction{reading, payment})
	require.NoError(t, err)
	require.Len(t, first, 2)
	require.NotEmpty(t, first[0].ID)

	repeat, err := store.InsertTransactions(ctx, []model.Transaction{reading, payment})
	require.NoError(t, err)
	require.Empty(t, repeat)

	correction, err := store.InsertTransactions(ctx, []model.Transaction{corrected})
	require.NoError(t, err)
	require.Empty(t, correction)

	require.Equal(t, 2, countOf(ctx, t, database, "transactions"))

	var amount float64

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &amount,
			`SELECT amount FROM transactions WHERE transaction_type_id = 125;`)
	}))
	require.InDelta(t, 11.50, amount, 0)
}

// GASACC-SC-017: A transaction keeps the date of Tbilisi that the gas service gave.
func TestTheStoreKeepsTheDateOfTbilisi(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	_, err := store.InsertTransactions(ctx, []model.Transaction{
		readingOf(customerC1, dateOf(time.September, 28), 1532, 11.33),
	})
	require.NoError(t, err)

	var date string

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &date, `SELECT transaction_date::text FROM transactions;`)
	}))
	require.Equal(t, "2026-09-28", date)
}

// GASACC-SC-009: A second assignment replaces the first, so a workspace holds one.
func TestTheStoreHoldsOneAssignment(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	_, err := store.AssignedCustomer(ctx)
	require.ErrorIs(t, err, errs.ErrNotAssigned)

	require.NoError(t, store.Assign(ctx, customerC1))
	require.NoError(t, store.Assign(ctx, customerC2))

	assigned, err := store.AssignedCustomer(ctx)
	require.NoError(t, err)
	require.Equal(t, customerC2, assigned)
	require.Equal(t, 1, countOf(ctx, t, database, "customer_assignments"))
}

// GASACC-SC-018: A new assignment and the end of the assignment keep the transactions
// of the customer before.
func TestTheTransactionsOutliveTheAssignment(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	require.NoError(t, store.Assign(ctx, customerC1))

	_, err := store.InsertTransactions(ctx, []model.Transaction{
		readingOf(customerC1, dateOf(time.September, 28), 1532, 11.33),
		paymentOf(dateOf(time.September, 1), 6.48),
	})
	require.NoError(t, err)

	require.NoError(t, store.Assign(ctx, customerC2))
	require.NoError(t, store.Unassign(ctx))

	require.Equal(t, 0, countOf(ctx, t, database, "customer_assignments"))
	require.Equal(t, 2, countOf(ctx, t, database, "transactions"))
}

// GASACC-SC-032: A transaction that acts in one workspace sees neither the assignment
// nor the transactions of another workspace.
func TestTheStoreServesNoOtherWorkspace(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	inW := actingIn(t, workspaceW)
	inX := actingIn(t, workspaceX)

	require.NoError(t, store.Assign(inW, customerC1))

	_, err := store.InsertTransactions(inW, []model.Transaction{
		readingOf(customerC1, dateOf(time.September, 28), 1532, 11.33),
	})
	require.NoError(t, err)

	_, err = store.AssignedCustomer(inX)
	require.ErrorIs(t, err, errs.ErrNotAssigned)

	_, err = store.LatestReading(inX)
	require.ErrorIs(t, err, errs.ErrNotAssigned)

	require.Equal(t, 0, countOf(inX, t, database, "customer_assignments"))
	require.Equal(t, 0, countOf(inX, t, database, "transactions"))
}

// GASACC-SC-025: The latest reading of the assigned customer skips a later payment, a
// later control reading, and the transactions of another customer (GASACC-DD-007).
func TestTheLatestReadingIsAReadingOfTheAssignedCustomer(t *testing.T) {
	t.Parallel()

	store, _ := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	require.NoError(t, store.Assign(ctx, customerC1))

	latest, err := store.LatestReading(ctx)
	require.NoError(t, err)
	require.Nil(t, latest)

	_, err = store.InsertTransactions(ctx, []model.Transaction{
		readingOf(customerC1, dateOf(time.August, 27), 1511, 6.48),
		readingOf(customerC1, dateOf(time.September, 28), 1532, 11.33),
		paymentOf(dateOf(time.October, 1), 11.33),
		{
			CustomerNumber: customerC1,
			TypeID:         operationControl,
			Operation:      "საკონტროლო ჩვენება",
			Date:           dateOf(time.October, 3),
			MeterValue:     1540,
		},
		readingOf(customerC2, dateOf(time.October, 5), 900, 3.10),
	})
	require.NoError(t, err)

	latest, err = store.LatestReading(ctx)
	require.NoError(t, err)
	require.NotNil(t, latest)
	require.Equal(t, "2026-09-28", latest.Date.Format(time.DateOnly))
	require.Equal(t, "რეალური ჩვენება", latest.Operation)
	require.InDelta(t, 1532.0, latest.MeterValue, 0)
	require.InDelta(t, 11.33, latest.Amount, 0)
}

// GASACC-SC-027: The pages hold the transactions of the assigned customer, the latest
// date first, with the identifier as the tie, and a page follows on from the one before.
func TestTheTransactionsPageRunsTheLatestFirst(t *testing.T) {
	t.Parallel()

	store, _ := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	require.NoError(t, store.Assign(ctx, customerC1))

	_, err := store.InsertTransactions(ctx, []model.Transaction{
		readingOf(customerC1, dateOf(time.August, 27), 1511, 6.48),
		paymentOf(dateOf(time.September, 1), 6.48),
		readingOf(customerC1, dateOf(time.September, 28), 1532, 11.33),
		paymentOf(dateOf(time.September, 28), 11.33),
		readingOf(customerC2, dateOf(time.October, 5), 900, 3.10),
	})
	require.NoError(t, err)

	first, err := store.TransactionsPage(ctx, page.Request{Limit: 3}, nil)
	require.NoError(t, err)
	require.Equal(t, []string{
		"2026-09-28/გაზის საფასურის ელ-გადახდა",
		"2026-09-28/რეალური ჩვენება",
		"2026-09-01/გაზის საფასურის ელ-გადახდა",
	}, datesOf(first.Items))
	require.NotNil(t, first.Next)
	require.Nil(t, first.Prev)

	second, err := store.TransactionsPage(ctx, page.Request{Limit: 3, Cursor: first.Next}, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"2026-08-27/რეალური ჩვენება"}, datesOf(second.Items))
	require.Nil(t, second.Next)
	require.NotNil(t, second.Prev)
}

// GASACC-SC-027: The filter from keeps the transactions of that date and after.
func TestTheTransactionsPageKeepsTheDatesFrom(t *testing.T) {
	t.Parallel()

	store, _ := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	require.NoError(t, store.Assign(ctx, customerC1))

	_, err := store.InsertTransactions(ctx, []model.Transaction{
		readingOf(customerC1, dateOf(time.August, 27), 1511, 6.48),
		paymentOf(dateOf(time.September, 1), 6.48),
	})
	require.NoError(t, err)

	from := dateOf(time.September, 1)

	window, err := store.TransactionsPage(ctx, page.Request{Limit: 20}, &from)
	require.NoError(t, err)
	require.Equal(t, []string{"2026-09-01/გაზის საფასურის ელ-გადახდა"}, datesOf(window.Items))
}

// GASACC-SC-030: The page of a workspace with no assigned customer is ErrNotAssigned.
func TestTheTransactionsPageNeedsTheAssignment(t *testing.T) {
	t.Parallel()

	store, _ := storeUnderTest(t)

	_, err := store.TransactionsPage(actingIn(t, workspaceW), page.Request{Limit: 20}, nil)
	require.ErrorIs(t, err, errs.ErrNotAssigned)
}
