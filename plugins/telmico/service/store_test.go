package service_test

import (
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/telmico/db"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
	"github.com/abgeo/maroid/plugins/telmico/service"
)

const (
	pluginID   = "dev.maroid.telmico"
	workspaceW = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"
	workspaceX = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e60"
	customerC2 = "8370999"

	operationCharge  = "დარიცხვა ჩვენების მიხედვით"
	operationPayment = "გადახდა"
)

// hubFixtures stands in for the core migrations of the hub: the workspaces that a
// scoped table references.
const hubFixtures = `
CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.workspaces (id) VALUES ('` + workspaceW + `'), ('` + workspaceX + `');`

// storeUnderTest starts a database, applies the migrations of the plugin, and returns
// the store and the handle that reads behind it.
func storeUnderTest(t *testing.T) (*service.Store, pluginapi.PluginDB) {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID(pluginID)
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	database := testdb.PluginDB(instance, identifier)

	return service.NewStore(database), database
}

func actingIn(t *testing.T, workspaceID string) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingWorkspace(t.Context(), workspaceID)
}

func dayOf(month time.Month, day int) time.Time {
	return time.Date(2026, month, day, 0, 0, 0, 0, model.Tbilisi())
}

func chargeOf(customer string, on time.Time, meter, usage, amount float64) model.Transaction {
	return model.Transaction{
		CustomerNumber: customer,
		Operation:      operationCharge,
		Date:           on,
		MeterValue:     new(meter),
		Usage:          new(usage),
		Amount:         new(amount),
		Balance:        new(amount),
		Tariff:         new(0.2405312),
		MeterSerial:    new("021242007390"),
	}
}

func paymentOf(customer string, on time.Time, amount float64) model.Transaction {
	return model.Transaction{
		CustomerNumber: customer,
		Operation:      operationPayment,
		Date:           on,
		Amount:         new(-amount),
		Balance:        new(0.0),
	}
}

func billOf(customer string, month model.Month) model.Bill {
	return model.Bill{
		CustomerNumber: customer,
		BillNumber:     customer + "-" + month.PortalForm(),
		ServiceBillID:  uuid.New(),
		Month:          month.First(),
		IssuedAt:       month.Next().First().Add(-14 * time.Hour),
		MeterSerial:    new("021242007390"),
	}
}

func countOf(ctx context.Context, t *testing.T, database pluginapi.PluginDB, table string) int {
	t.Helper()

	var count int

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count, `SELECT count(*) FROM `+table+`;`)
	}))

	return count
}

// ELECACC-SC-016: Two charges of one date with two meter values are two transactions.
// A repeat stores nothing, and a corrected amount replaces the amount of its charge.
func TestTheStoreKeepsOneTransactionOfAKindOnADateWithAMeterValue(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	low := chargeOf(customerC1, dayOf(time.April, 28), 1163, 13.2, 2.52)
	high := chargeOf(customerC1, dayOf(time.April, 28), 1317, 153.3, 36.87)
	payment := paymentOf(customerC1, dayOf(time.April, 1), 48.57)

	first, err := store.InsertTransactions(ctx, []model.Transaction{low, high, payment})
	require.NoError(t, err)
	require.Len(t, first, 3)
	assert.NotEmpty(t, first[0].ID)

	repeat, err := store.InsertTransactions(ctx, []model.Transaction{low, high, payment})
	require.NoError(t, err)
	assert.Empty(t, repeat)

	corrected := chargeOf(customerC1, dayOf(time.April, 28), 1317, 153.3, 36.90)

	correction, err := store.InsertTransactions(ctx, []model.Transaction{corrected})
	require.NoError(t, err)
	assert.Empty(t, correction)

	assert.Equal(t, 3, countOf(ctx, t, database, "transactions"))

	charges, err := store.ChargesOn(ctx, customerC1, dayOf(time.April, 28))
	require.NoError(t, err)
	require.Len(t, charges, 2)

	reading := model.ReadingsOf(charges)[0]
	assert.InDelta(t, 166.5, reading.Usage, 1e-9)
	assert.InDelta(t, 39.42, reading.Charge, 1e-9)
	assert.InDelta(t, 1317.0, reading.MeterValue, 0)
}

// ELECACC-SC-017: One bill number is one bill, with the first day of its month.
func TestTheStoreKeepsOneBillForABillNumber(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	september := model.Month{Year: 2026, Month: time.September}
	bill := billOf(customerC1, september)

	require.NoError(t, store.InsertBills(ctx, []model.Bill{bill}))
	require.NoError(t, store.InsertBills(ctx, []model.Bill{bill}))

	assert.Equal(t, 1, countOf(ctx, t, database, "bills"))

	held, err := store.BillByMonth(ctx, customerC1, september)
	require.NoError(t, err)
	require.NotNil(t, held)
	assert.Equal(t, "8370764-2026-9", held.BillNumber)
	assert.Equal(t, bill.ServiceBillID, held.ServiceBillID)
	assert.Equal(t, "2026-09-01", held.Month.Format(time.DateOnly))

	absent, err := store.BillByMonth(ctx, customerC1, september.Next())
	require.NoError(t, err)
	assert.Nil(t, absent)
}

// ELECACC-SC-010: A second assignment replaces the first, so the workspace holds one.
func TestTheWorkspaceHoldsOneAssignment(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	_, err := store.AssignedCustomer(ctx)
	require.ErrorIs(t, err, errs.ErrNotAssigned)

	require.NoError(t, store.Assign(ctx, customerC1))
	require.NoError(t, store.Assign(ctx, customerC2))

	assigned, err := store.AssignedCustomer(ctx)
	require.NoError(t, err)
	assert.Equal(t, customerC2, assigned)
	assert.Equal(t, 1, countOf(ctx, t, database, "customer_assignments"))
}

// ELECACC-SC-018: A new assignment and the removal of the assignment keep the
// transactions and the bills of the customer assigned before.
func TestTheHistoryOutlivesTheAssignment(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	require.NoError(t, store.Assign(ctx, customerC1))

	_, err := store.InsertTransactions(ctx, []model.Transaction{
		chargeOf(customerC1, dayOf(time.September, 28), 1989, 162.6, 39.11),
	})
	require.NoError(t, err)
	require.NoError(t, store.InsertBills(ctx, []model.Bill{
		billOf(customerC1, model.Month{Year: 2026, Month: time.September}),
	}))

	require.NoError(t, store.Assign(ctx, customerC2))
	require.NoError(t, store.Unassign(ctx))

	assert.Equal(t, 1, countOf(ctx, t, database, "transactions"))
	assert.Equal(t, 1, countOf(ctx, t, database, "bills"))
}

// ELECACC-SC-023: The latest reading holds every charge of the latest date of a charge.
// A later payment and the first reading of a new meter are no reading.
func TestTheLatestReadingHoldsTheChargesOfItsDate(t *testing.T) {
	t.Parallel()

	store, _ := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	_, err := store.LatestReading(ctx)
	require.ErrorIs(t, err, errs.ErrNotAssigned)

	require.NoError(t, store.Assign(ctx, customerC1))

	charges, err := store.LatestReading(ctx)
	require.NoError(t, err)
	assert.Empty(t, charges)

	newMeter := model.Transaction{
		CustomerNumber: customerC1,
		Operation:      "მრიცხველის დადგმის ჩვენება",
		Date:           dayOf(time.May, 2),
		MeterValue:     new(3.0),
		Usage:          new(0.0),
	}

	_, err = store.InsertTransactions(ctx, []model.Transaction{
		chargeOf(customerC1, dayOf(time.March, 29), 1150, 254.9, 48.57),
		chargeOf(customerC1, dayOf(time.April, 28), 1163, 13.2, 2.52),
		chargeOf(customerC1, dayOf(time.April, 28), 1317, 153.3, 36.87),
		paymentOf(customerC1, dayOf(time.May, 1), 39.39),
		newMeter,
		chargeOf(customerC2, dayOf(time.June, 28), 1, 1, 1),
	})
	require.NoError(t, err)

	charges, err = store.LatestReading(ctx)
	require.NoError(t, err)
	require.Len(t, charges, 2)

	reading := model.ReadingsOf(charges)[0]
	assert.Equal(t, "2026-04-28", reading.Date.Format(time.DateOnly))
	assert.InDelta(t, 166.5, reading.Usage, 1e-9)
	assert.InDelta(t, 39.39, reading.Charge, 1e-9)
}

// ELECACC-SC-025: The page of the transactions holds the assigned customer alone, the
// latest date first, and a next link while more remain.
func TestTheTransactionsPageReadsTheLatestFirst(t *testing.T) {
	t.Parallel()

	store, _ := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	require.NoError(t, store.Assign(ctx, customerC1))

	var transactions []model.Transaction
	for month := time.January; month <= time.September; month++ {
		transactions = append(transactions,
			chargeOf(customerC1, dayOf(month, 28), float64(month)*100, 100, 20),
			paymentOf(customerC1, dayOf(month, 1), 20),
		)
	}

	transactions = append(
		transactions,
		chargeOf(customerC2, dayOf(time.October, 28), 1, 1, 1),
	)

	_, err := store.InsertTransactions(ctx, transactions)
	require.NoError(t, err)

	first, err := store.TransactionsPage(ctx, page.Request{Limit: 10}, nil)
	require.NoError(t, err)
	require.Len(t, first.Items, 10)
	assert.Equal(t, "2026-09-28", first.Items[0].Date.Format(time.DateOnly))
	assert.Equal(t, "2026-09-01", first.Items[1].Date.Format(time.DateOnly))
	require.NotNil(t, first.Next)

	second, err := store.TransactionsPage(ctx, page.Request{Limit: 10, Cursor: first.Next}, nil)
	require.NoError(t, err)
	assert.Len(t, second.Items, 8)
	assert.Nil(t, second.Next)

	from := dayOf(time.September, 1)

	recent, err := store.TransactionsPage(ctx, page.Request{Limit: 10}, &from)
	require.NoError(t, err)
	assert.Len(t, recent.Items, 2)
}

// ELECACC-SC-027: The page of the bills holds the assigned customer alone, the latest
// month first.
func TestTheBillsPageReadsTheLatestMonthFirst(t *testing.T) {
	t.Parallel()

	store, _ := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	_, err := store.BillsPage(ctx, page.Request{Limit: 20})
	require.ErrorIs(t, err, errs.ErrNotAssigned)

	require.NoError(t, store.Assign(ctx, customerC1))

	months := model.Period{
		From: model.Month{Year: 2024, Month: time.October},
		To:   september,
	}.Months()
	bills := make([]model.Bill, 0, len(months)+1)

	for _, month := range months {
		bills = append(bills, billOf(customerC1, month))
	}

	bills = append(bills, billOf(customerC2, september))
	require.NoError(t, store.InsertBills(ctx, bills))

	first, err := store.BillsPage(ctx, page.Request{Limit: 20})
	require.NoError(t, err)
	require.Len(t, first.Items, 20)
	assert.Equal(t, "8370764-2026-9", first.Items[0].BillNumber)
	require.NotNil(t, first.Next)

	second, err := store.BillsPage(ctx, page.Request{Limit: 20, Cursor: first.Next})
	require.NoError(t, err)
	assert.Len(t, second.Items, 4)

	held, err := store.Bill(ctx, first.Items[0].ID)
	require.NoError(t, err)
	assert.Equal(t, "8370764-2026-9", held.BillNumber)

	_, err = store.Bill(ctx, uuid.NewString())
	require.ErrorIs(t, err, errs.ErrBillUnknown)
}

// ELECACC-SC-029: A bill of a customer that the workspace no longer assigns is no bill
// that a member downloads.
func TestABillOfAnotherCustomerIsUnknown(t *testing.T) {
	t.Parallel()

	store, _ := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	require.NoError(t, store.Assign(ctx, customerC1))
	require.NoError(t, store.InsertBills(ctx, []model.Bill{
		billOf(customerC1, model.Month{Year: 2026, Month: time.September}),
	}))

	window, err := store.BillsPage(ctx, page.Request{Limit: 20})
	require.NoError(t, err)
	require.Len(t, window.Items, 1)

	require.NoError(t, store.Assign(ctx, customerC2))

	_, err = store.Bill(ctx, window.Items[0].ID)
	require.ErrorIs(t, err, errs.ErrBillUnknown)
}

// ELECACC-SC-032: The assignment, the transactions, and the bills of one workspace
// are not visible in another.
func TestAWorkspaceSeesNothingOfAnother(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	inW := actingIn(t, workspaceW)
	inX := actingIn(t, workspaceX)

	require.NoError(t, store.Assign(inW, customerC1))
	_, err := store.InsertTransactions(inW, []model.Transaction{
		chargeOf(customerC1, dayOf(time.September, 28), 1989, 162.6, 39.11),
	})
	require.NoError(t, err)
	require.NoError(t, store.InsertBills(inW, []model.Bill{
		billOf(customerC1, model.Month{Year: 2026, Month: time.September}),
	}))

	_, err = store.AssignedCustomer(inX)
	require.ErrorIs(t, err, errs.ErrNotAssigned)

	assert.Equal(t, 0, countOf(inX, t, database, "transactions"))
	assert.Equal(t, 0, countOf(inX, t, database, "bills"))
	assert.Equal(t, 0, countOf(inX, t, database, "customer_assignments"))
}
