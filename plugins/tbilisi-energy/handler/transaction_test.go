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
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/db"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/handler"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/service"
)

const (
	pathTransactionList = "/transactions"
	externalURL         = "https://hub.example.ge"
	fileBase            = "https://my.te.ge/"

	operationReading = 125
	operationPayment = 196
)

// transactionPage is the answer of one page of transactions.
type transactionPage struct {
	Items []transactionItem `json:"items"`
	Next  *string           `json:"next"`
	Prev  *string           `json:"prev"`
}

type transactionItem struct {
	ID              string   `json:"id"`
	TransactionDate string   `json:"transaction_date"`
	Operation       string   `json:"operation"`
	Amount          money    `json:"amount"`
	Balance         money    `json:"balance"`
	Usage           *float64 `json:"usage"`
	MeterValue      *float64 `json:"meter_value"`
	BillURL         *string  `json:"bill_url"`
	MeterPhotoURL   *string  `json:"meter_photo_url"`
}

// money is the money object of Z-173.
type money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// transactionsUnderTest migrates a database, and answers the store of workspace W and
// the route of the transactions over it, as the hub mounts the route.
func transactionsUnderTest(t *testing.T) (*service.Store, http.Handler) {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(`CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
		INSERT INTO public.workspaces (id) VALUES ('` + workspaceW + `');`)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID("dev.maroid.tbilisi-energy")
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	store := service.NewStore(testdb.PluginDB(instance, identifier))
	routes := handler.NewTransactionHandler(slog.New(slog.DiscardHandler), store, fileBase).Routes()
	require.Len(t, routes, 1)

	actingInWorkspace := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(
				w,
				r.WithContext(pluginapi.ContextWithActingWorkspace(r.Context(), workspaceW)),
			)
		})
	}

	return store, address.Middleware(externalURL)(actingInWorkspace(routes[0].Handler))
}

func actingInW() context.Context {
	return pluginapi.ContextWithActingWorkspace(context.Background(), workspaceW)
}

// monthsOf answers a reading on the 28th and a payment on the 1st of each month from
// the first month on, of the customer.
func monthsOf(customer string, first time.Time, months int) []model.Transaction {
	transactions := make([]model.Transaction, 0, 2*months)

	for month := range months {
		start := first.AddDate(0, month, 0)
		bill := "api/Customer/GetCustomerBillPdf/" + start.Format("2006-01") + ".pdf"

		transactions = append(transactions,
			model.Transaction{
				CustomerNumber: customer,
				TypeID:         operationPayment,
				Operation:      "გაზის საფასურის ელ-გადახდა",
				Date:           start,
				Amount:         10,
			},
			model.Transaction{
				CustomerNumber: customer,
				TypeID:         operationReading,
				Operation:      "რეალური ჩვენება",
				Date:           start.AddDate(0, 0, 27),
				Consumption:    20,
				MeterValue:     1000 + 20*float64(month),
				Amount:         10,
				Balance:        10,
				BillPath:       &bill,
			},
		)
	}

	return transactions
}

func firstOf(year int, month time.Month) time.Time {
	return time.Date(year, month, 1, 0, 0, 0, 0, model.Tbilisi())
}

func get(t *testing.T, route http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	route.ServeHTTP(recorder, request)

	return recorder
}

func pageOf(t *testing.T, recorder *httptest.ResponseRecorder) transactionPage {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var answer transactionPage
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))

	return answer
}

func pathOf(t *testing.T, link string) string {
	t.Helper()

	parsed, err := url.Parse(link)
	require.NoError(t, err)

	return parsed.RequestURI()
}

// GASACC-SC-027: The first page holds the 20 latest transactions of the assigned
// customer, the next page the 20 after them, and no transaction of another customer.
// A reading carries its usage and its meter value, and a payment carries neither. An
// amount is the money object (GASACC-DD-015).
func TestTheTransactionsPageTheLatestFirst(t *testing.T) {
	t.Parallel()

	store, route := transactionsUnderTest(t)
	ctx := actingInW()

	require.NoError(t, store.Assign(ctx, customerC1))

	held := slices.Concat(
		monthsOf(customerC1, firstOf(2024, time.October), 23)[:45],
		monthsOf(customerC2, firstOf(2026, time.August), 2)[:3],
	)
	_, err := store.InsertTransactions(ctx, held)
	require.NoError(t, err)

	first := pageOf(t, get(t, route, pathTransactionList))
	require.Len(t, first.Items, 20)
	require.NotNil(t, first.Next)
	require.Nil(t, first.Prev)

	latest := first.Items[0]
	require.Equal(t, "2026-08-01", latest.TransactionDate)
	require.Equal(t, "გაზის საფასურის ელ-გადახდა", latest.Operation)
	require.Nil(t, latest.Usage)
	require.Nil(t, latest.MeterValue)
	require.Nil(t, latest.BillURL)

	require.Equal(t, money{Amount: 10, Currency: "GEL"}, latest.Amount)
	require.Equal(t, money{Amount: 0, Currency: "GEL"}, latest.Balance)

	reading := first.Items[1]
	require.Equal(t, "2026-07-28", reading.TransactionDate)
	require.NotNil(t, reading.Usage)
	require.InDelta(t, 20.0, *reading.Usage, 0)
	require.NotNil(t, reading.MeterValue)

	second := pageOf(t, get(t, route, pathOf(t, *first.Next)))
	require.Len(t, second.Items, 20)
	require.Less(t, second.Items[0].TransactionDate, first.Items[19].TransactionDate)

	for _, page := range []transactionPage{first, second} {
		for _, item := range page.Items {
			require.Less(t, item.TransactionDate, "2026-08-02")
		}
	}
}

// GASACC-SC-028: A reading links to its bill at the gas service, and to no meter photo
// that the gas service does not hold. A payment links to neither.
func TestATransactionLinksToItsFiles(t *testing.T) {
	t.Parallel()

	store, route := transactionsUnderTest(t)
	ctx := actingInW()

	require.NoError(t, store.Assign(ctx, customerC1))

	_, err := store.InsertTransactions(ctx, monthsOf(customerC1, firstOf(2026, time.September), 1))
	require.NoError(t, err)

	answer := pageOf(t, get(t, route, pathTransactionList))
	require.Len(t, answer.Items, 2)

	reading, payment := answer.Items[0], answer.Items[1]
	require.NotNil(t, reading.BillURL)
	require.Equal(
		t,
		"https://my.te.ge/api/Customer/GetCustomerBillPdf/2026-09.pdf",
		*reading.BillURL,
	)
	require.Nil(t, reading.MeterPhotoURL)
	require.Nil(t, payment.BillURL)
	require.Nil(t, payment.MeterPhotoURL)
}

// GASACC-SC-027: A control reading answers its meter value and no usage
// (GASACC-DD-012).
func TestAControlReadingAnswersItsMeterValue(t *testing.T) {
	t.Parallel()

	store, route := transactionsUnderTest(t)
	ctx := actingInW()

	require.NoError(t, store.Assign(ctx, customerC1))

	_, err := store.InsertTransactions(ctx, []model.Transaction{{
		CustomerNumber: customerC1,
		TypeID:         172,
		Operation:      "საკონტროლო ჩვენება",
		Date:           firstOf(2026, time.September).AddDate(0, 0, 16),
		MeterValue:     982,
	}})
	require.NoError(t, err)

	answer := pageOf(t, get(t, route, pathTransactionList))
	require.Len(t, answer.Items, 1)
	require.Nil(t, answer.Items[0].Usage)
	require.NotNil(t, answer.Items[0].MeterValue)
	require.InDelta(t, 982.0, *answer.Items[0].MeterValue, 0)
}

// GASACC-SC-030: A workspace with no assigned customer answers not-assigned, and one
// with no transaction answers an empty page.
func TestTheTransactionsNeedTheAssignment(t *testing.T) {
	t.Parallel()

	store, route := transactionsUnderTest(t)

	recorder := get(t, route, pathTransactionList)
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, problemPrefix+"not-assigned", problemTypeOf(t, recorder))

	require.NoError(t, store.Assign(actingInW(), customerC1))

	answer := pageOf(t, get(t, route, pathTransactionList))
	require.Empty(t, answer.Items)
}

// GASACC-SC-027: The filter from keeps the transactions of that date and after, and a
// date that does not parse is refused.
func TestTheTransactionsFilterFrom(t *testing.T) {
	t.Parallel()

	store, route := transactionsUnderTest(t)
	ctx := actingInW()

	require.NoError(t, store.Assign(ctx, customerC1))

	_, err := store.InsertTransactions(ctx, monthsOf(customerC1, firstOf(2026, time.August), 2))
	require.NoError(t, err)

	answer := pageOf(t, get(t, route, pathTransactionList+"?from=2026-09-01"))
	require.Len(t, answer.Items, 2)

	recorder := get(t, route, pathTransactionList+"?from=01/09/2026")
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
