package handler_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/telmico/db"
	"github.com/abgeo/maroid/plugins/telmico/handler"
	"github.com/abgeo/maroid/plugins/telmico/model"
	"github.com/abgeo/maroid/plugins/telmico/service"
)

const (
	externalURL = "https://hub.example.ge"

	operationCharge  = "დარიცხვა ჩვენების მიხედვით"
	operationPayment = "გადახდა"
	operationMeter   = "მრიცხველის დადგმის ჩვენება"

	currencyLari = "GEL"
)

// money is the money object of Z-173.
type money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type transactionItem struct {
	ID              string   `json:"id"`
	TransactionDate string   `json:"transaction_date"`
	Operation       string   `json:"operation"`
	Amount          *money   `json:"amount"`
	Balance         *money   `json:"balance"`
	Usage           *float64 `json:"usage"`
	MeterValue      *float64 `json:"meter_value"`
	Tariff          *float64 `json:"tariff"`
}

type billItem struct {
	ID         string `json:"id"`
	Month      string `json:"month"`
	BillNumber string `json:"bill_number"`
	IssuedOn   string `json:"issued_on"`
	Document   string `json:"document"`
}

type pageOf[T any] struct {
	Items []T     `json:"items"`
	Next  *string `json:"next"`
	Prev  *string `json:"prev"`
}

// listsUnderTest migrates a database, and answers the store of workspace W and the list
// routes over it, mounted as the hub mounts them.
func listsUnderTest(t *testing.T) (*service.Store, http.Handler) {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(`CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
		INSERT INTO public.workspaces (id) VALUES ('` + workspaceW + `');`)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID("dev.maroid.telmico")
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	store := service.NewStore(testdb.PluginDB(instance, identifier))
	logger := slog.New(slog.DiscardHandler)

	routes := slices.Concat(
		handler.NewTransactionHandler(logger, store).Routes(),
		handler.NewBillHandler(logger, store, nil).Routes(),
	)

	router := chi.NewRouter()

	for _, route := range routes {
		router.Method(route.Method, route.Pattern, route.Handler)
	}

	actingInWorkspace := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(
			w,
			r.WithContext(pluginapi.ContextWithActingWorkspace(r.Context(), workspaceW)),
		)
	})

	return store, address.Middleware(externalURL)(actingInWorkspace)
}

func actingInW() context.Context {
	return pluginapi.ContextWithActingWorkspace(context.Background(), workspaceW)
}

func get(t *testing.T, route http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	route.ServeHTTP(recorder, request)

	return recorder
}

func decodePage[T any](t *testing.T, recorder *httptest.ResponseRecorder) pageOf[T] {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var answer pageOf[T]
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))

	return answer
}

func pathOf(t *testing.T, link string) string {
	t.Helper()

	parsed, err := url.Parse(link)
	require.NoError(t, err)

	return parsed.RequestURI()
}

// monthsOf answers a charge on the 28th and a payment on the 1st of each month of 2025
// and of 2026 up to the month, then the first reading of a new meter, 45 rows in all.
func historyOf(customer string) []model.Transaction {
	var transactions []model.Transaction

	for month := range 22 {
		first := time.Date(2025, time.January, 1, 0, 0, 0, 0, model.Tbilisi()).AddDate(0, month, 0)

		charge := chargeOf(customer)
		charge.Date = first.AddDate(0, 0, 27)
		charge.MeterValue = new(1000 + 100*float64(month))

		transactions = append(transactions, charge, model.Transaction{
			CustomerNumber: customer,
			Operation:      operationPayment,
			Date:           first,
			Amount:         new(-39.11),
			Balance:        new(0.0),
		})
	}

	return append(transactions, model.Transaction{
		CustomerNumber: customer,
		Operation:      operationMeter,
		Date:           time.Date(2024, time.October, 2, 0, 0, 0, 0, model.Tbilisi()),
		MeterValue:     new(3.0),
		Usage:          new(0.0),
	})
}

// ELECACC-SC-025: The first page holds the 20 latest transactions of the assigned
// customer, and the next link leads on. A charge carries every value, a payment its
// amount and its balance, and the first reading of a new meter its meter value.
func TestTheTransactionsPageTheLatestFirst(t *testing.T) {
	t.Parallel()

	store, route := listsUnderTest(t)
	ctx := actingInW()

	require.NoError(t, store.Assign(ctx, customerC1))
	_, err := store.InsertTransactions(ctx, historyOf(customerC1))
	require.NoError(t, err)

	first := decodePage[transactionItem](t, get(t, route, "/transactions"))
	require.Len(t, first.Items, 20)
	assert.Equal(t, "2026-10-28", first.Items[0].TransactionDate)
	require.NotNil(t, first.Next)
	assert.True(t, strings.HasPrefix(*first.Next, externalURL+"/transactions?"))

	charge := first.Items[0]
	assert.Equal(t, operationCharge, charge.Operation)
	assert.Equal(t, &money{Amount: 39.11, Currency: currencyLari}, charge.Amount)
	assert.Equal(t, &money{Amount: 39.11, Currency: currencyLari}, charge.Balance)
	assert.Equal(t, new(162.6), charge.Usage)
	assert.Equal(t, new(3100.0), charge.MeterValue)
	assert.Equal(t, new(0.2405312), charge.Tariff)

	payment := first.Items[1]
	assert.Equal(t, &money{Amount: -39.11, Currency: currencyLari}, payment.Amount)
	assert.NotNil(t, payment.Balance)
	assert.Nil(t, payment.Usage)
	assert.Nil(t, payment.MeterValue)
	assert.Nil(t, payment.Tariff)

	third := decodePage[transactionItem](
		t,
		get(
			t,
			route,
			pathOf(t, *decodePage[transactionItem](t, get(t, route, pathOf(t, *first.Next))).Next),
		),
	)
	require.Len(t, third.Items, 5)

	newMeter := third.Items[4]
	assert.Equal(t, operationMeter, newMeter.Operation)
	assert.Equal(t, new(3.0), newMeter.MeterValue)
	assert.Nil(t, newMeter.Amount)
	assert.Nil(t, newMeter.Balance)
	assert.Nil(t, newMeter.Usage)
	assert.Nil(t, third.Next)
}

// ELECACC-SC-025: The filter from keeps the transactions of that date and after, and a
// date that does not parse answers 400.
func TestTheTransactionsFilterFrom(t *testing.T) {
	t.Parallel()

	store, route := listsUnderTest(t)
	ctx := actingInW()

	require.NoError(t, store.Assign(ctx, customerC1))
	_, err := store.InsertTransactions(ctx, historyOf(customerC1))
	require.NoError(t, err)

	recent := decodePage[transactionItem](
		t,
		get(t, route, "/transactions?from=2026-10-01&limit=100"),
	)
	assert.Len(t, recent.Items, 2)

	assert.Equal(t, http.StatusBadRequest, get(t, route, "/transactions?from=01.10.2026").Code)
}

// ELECACC-SC-027: The first page holds 20 bills of the assigned customer, the latest
// month first, each with the absolute address of its document.
func TestTheBillsPageTheLatestMonthFirst(t *testing.T) {
	t.Parallel()

	store, route := listsUnderTest(t)
	ctx := actingInW()

	require.NoError(t, store.Assign(ctx, customerC1))

	months := model.Period{
		From: model.Month{Year: 2024, Month: time.October},
		To:   model.Month{Year: 2026, Month: time.September},
	}.Months()
	bills := make([]model.Bill, 0, len(months))

	for _, month := range months {
		bills = append(bills, model.Bill{
			CustomerNumber: customerC1,
			BillNumber:     customerC1 + "-" + month.PortalForm(),
			ServiceBillID:  uuid.New(),
			Month:          month.First(),
			IssuedAt:       month.Next().First().Add(-14 * time.Hour),
		})
	}

	require.NoError(t, store.InsertBills(ctx, bills))

	first := decodePage[billItem](t, get(t, route, "/bills"))
	require.Len(t, first.Items, 20)
	require.NotNil(t, first.Next)

	latest := first.Items[0]
	assert.Equal(t, "2026-09", latest.Month)
	assert.Equal(t, "8370764-2026-9", latest.BillNumber)
	assert.Equal(t, "2026-09-30", latest.IssuedOn)
	assert.Equal(t, externalURL+"/bills/"+latest.ID+"/document", latest.Document)

	second := decodePage[billItem](t, get(t, route, pathOf(t, *first.Next)))
	assert.Len(t, second.Items, 4)
}

// ELECACC-SC-030: With no assigned customer both lists answer not-assigned. With an
// assigned customer and no row, both answer an empty page.
func TestTheListsNeedTheAssignment(t *testing.T) {
	t.Parallel()

	store, route := listsUnderTest(t)

	for _, target := range []string{"/transactions", "/bills"} {
		recorder := get(t, route, target)

		assert.Equal(t, http.StatusConflict, recorder.Code)
		assert.Equal(t, problemPrefix+"not-assigned", problemTypeOf(t, recorder))
	}

	require.NoError(t, store.Assign(actingInW(), customerC1))

	assert.Empty(t, decodePage[transactionItem](t, get(t, route, "/transactions")).Items)
	assert.Empty(t, decodePage[billItem](t, get(t, route, "/bills")).Items)
}
