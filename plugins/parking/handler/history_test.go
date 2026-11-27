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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/parking/db"
	"github.com/abgeo/maroid/plugins/parking/handler"
	"github.com/abgeo/maroid/plugins/parking/model"
	"github.com/abgeo/maroid/plugins/parking/service"
)

const (
	userB    = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e79"
	plateA   = "AB071EO"
	plateB   = "CD123EF"
	lotA1637 = "A1637"

	parameterFree = "free"
)

// historyRig runs the history routes over a database that holds the migrations of the
// plugin.
type historyRig struct {
	store  *service.HistoryStore
	routes []pluginapi.Route
}

func newHistoryRig(t *testing.T) *historyRig {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(`CREATE TABLE public.users (id UUID NOT NULL PRIMARY KEY);
		INSERT INTO public.users (id) VALUES ('` + userA + `'), ('` + userB + `');`)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID("dev.maroid.parking")
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	store := service.NewHistoryStore(testdb.PluginDB(instance, identifier))

	return &historyRig{store: store, routes: historyRoutes(store)}
}

func historyRoutes(store *service.HistoryStore) []pluginapi.Route {
	logger := slog.New(slog.DiscardHandler)

	return slices.Concat(
		handler.NewSessionHandler(logger, store).Routes(),
		handler.NewTransactionHandler(logger, store).Routes(),
	)
}

// actingIn answers the context of the user in workspace W, in Asia/Tbilisi.
func actingIn(t *testing.T, user string) context.Context {
	t.Helper()

	return actingFrom(context.Background(), t, user)
}

// actingFrom adds the user in workspace W, in Asia/Tbilisi, to the context.
func actingFrom(ctx context.Context, t *testing.T, user string) context.Context {
	t.Helper()

	zone, err := time.LoadLocation("Asia/Tbilisi")
	require.NoError(t, err)

	return pluginapi.ContextWithActingTimeZone(
		pluginapi.ContextWithActingWorkspace(
			pluginapi.ContextWithActingUser(ctx, user), workspaceW,
		),
		zone,
	)
}

func (h *historyRig) save(
	t *testing.T,
	user string,
	sessions []model.Session,
	transactions []model.Transaction,
) {
	t.Helper()

	_, err := h.store.Save(actingIn(t, user), sessions, transactions)
	require.NoError(t, err)
}

// get answers the route of the target, read as user A.
func (h *historyRig) get(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()

	return serve(t, h.routes, target)
}

// serve answers the route of the target, read as user A.
func serve(t *testing.T, routes []pluginapi.Route, target string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(actingIn(t, userA), http.MethodGet, target, nil)

	for _, route := range routes {
		if route.Method == http.MethodGet && route.Pattern == request.URL.Path {
			recorder := httptest.NewRecorder()
			address.Middleware("https://maroid.test")(route.Handler).ServeHTTP(recorder, request)

			return recorder
		}
	}

	t.Fatalf("no route GET %s", request.URL.Path)

	return nil
}

// historyPage is the answer of a page of a collection.
type historyPage[T any] struct {
	Items []T     `json:"items"`
	Next  *string `json:"next"`
	Prev  *string `json:"prev"`
}

func pageOf[T any](t *testing.T, recorder *httptest.ResponseRecorder) historyPage[T] {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var answer historyPage[T]
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))

	return answer
}

func follow(t *testing.T, link *string) string {
	t.Helper()

	require.NotNil(t, link)

	parsed, err := url.Parse(*link)
	require.NoError(t, err)

	return parsed.RequestURI()
}

type money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type sessionAnswer struct {
	ID              string    `json:"id"`
	LotNumber       string    `json:"lot_number"`
	Address         string    `json:"address"`
	Plate           string    `json:"plate"`
	StartedAt       time.Time `json:"started_at"`
	EndedAt         time.Time `json:"ended_at"`
	DurationSeconds int64     `json:"duration_seconds"`
	Amount          money     `json:"amount"`
	Free            bool      `json:"free"`
}

type transactionAnswer struct {
	ID           string     `json:"id"`
	OccurredAt   time.Time  `json:"occurred_at"`
	Kind         string     `json:"kind"`
	Description  string     `json:"description"`
	Amount       money      `json:"amount"`
	BalanceAfter money      `json:"balance_after"`
	Session      *lotAnswer `json:"session"`
}

type lotAnswer struct {
	LotNumber string `json:"lot_number"`
	Address   string `json:"address"`
}

type sessionTotal struct {
	Count           int   `json:"count"`
	DurationSeconds int64 `json:"duration_seconds"`
	Amount          money `json:"amount"`
}

type transactionTotal struct {
	Count   int   `json:"count"`
	TopUps  money `json:"top_ups"`
	Charges money `json:"charges"`
}

func totalOf[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var answer T
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))

	return answer
}

func sessionAt(
	id int64,
	lot string,
	plate string,
	started time.Time,
	amount float64,
) model.Session {
	return model.Session{
		ExternalID: id, LotNumber: lot, Address: "Street of " + lot, Plate: plate,
		ParkingType: "BOTH", StartedAt: started, EndedAt: started.Add(30 * time.Minute),
		Amount: amount,
	}
}

func transactionAt(
	id int64,
	kind model.TransactionKind,
	occurred time.Time,
	amount float64,
	balance float64,
) model.Transaction {
	return model.Transaction{
		ExternalID:   id,
		Kind:         kind,
		ServiceKind:  string(kind),
		Amount:       amount,
		BalanceAfter: balance,
		OccurredAt:   occurred,
		Description:  "Row " + strconv.FormatInt(id, 10),
	}
}

func day(month time.Month, date int, hour int) time.Time {
	return time.Date(2026, month, date, hour, 0, 0, 0, time.UTC)
}

// PARKHIST-SC-022: Each month sums its charges and holds the balance of its latest
// transaction, in the time zone of the user. No transaction of another user counts.
func TestTheChargeMonthsFollowTheZoneOfTheUser(t *testing.T) {
	t.Parallel()

	rig := newHistoryRig(t)
	rig.save(t, userA, nil, []model.Transaction{
		transactionAt(1, model.TransactionKindTopUp, day(time.January, 5, 9), 10, 10),
		transactionAt(2, model.TransactionKindParkingPayment, day(time.January, 20, 9), 1.07, 8.93),
		transactionAt(3, model.TransactionKindParkingPayment,
			time.Date(2026, time.January, 31, 21, 30, 0, 0, time.UTC), 0.55, 8.38),
	})
	rig.save(t, userB, nil, []model.Transaction{
		transactionAt(9, model.TransactionKindParkingPayment, day(time.January, 10, 9), 40, 1),
	})

	type month struct {
		Month   string `json:"month"`
		Charges money  `json:"charges"`
		Balance money  `json:"balance"`
	}

	answer := pageOf[month](t, rig.get(t, "/charge-months?from=2026-01-01"))

	require.Len(t, answer.Items, 2)
	assert.Equal(t, "2026-01", answer.Items[0].Month)
	assert.InDelta(t, 1.07, answer.Items[0].Charges.Amount, 1e-9)
	assert.InDelta(t, 8.93, answer.Items[0].Balance.Amount, 1e-9)
	assert.Equal(t, "2026-02", answer.Items[1].Month)
	assert.InDelta(t, 0.55, answer.Items[1].Charges.Amount, 1e-9)
	assert.Nil(t, answer.Next)
}

// PARKHIST-SC-023: The overview lists the 5 lots with the most sessions. A tie goes to the
// later last session.
func TestTheLotUsagesHoldTheFiveBusiestLots(t *testing.T) {
	t.Parallel()

	rig := newHistoryRig(t)

	var sessions []model.Session

	id := int64(1)
	add := func(lot string, date int) {
		sessions = append(sessions, sessionAt(id, lot, plateA, day(time.March, date, 9), 0.50))
		id++
	}

	for date := 1; date <= 4; date++ {
		add(lotA1637, date)
	}

	add("A894", 5)
	add("A894", 6)
	add("A203", 7)
	add("A203", 8)

	for _, lot := range []string{"B1", "B2", "B3", "B4"} {
		add(lot, 2)
	}

	rig.save(t, userA, sessions, nil)

	type usage struct {
		LotNumber string `json:"lot_number"`
		Address   string `json:"address"`
		Count     int    `json:"count"`
		Amount    money  `json:"amount"`
	}

	answer := pageOf[usage](t, rig.get(t, "/lot-usages"))

	require.Len(t, answer.Items, 5)
	assert.Equal(t, lotA1637, answer.Items[0].LotNumber)
	assert.Equal(t, 4, answer.Items[0].Count)
	assert.InDelta(t, 2.00, answer.Items[0].Amount.Amount, 1e-9)
	assert.Equal(t, "A203", answer.Items[1].LotNumber)
	assert.Equal(t, "A894", answer.Items[2].LotNumber)
	assert.Equal(t, 1, answer.Items[3].Count)
	assert.Equal(t, 1, answer.Items[4].Count)
}

// PARKHIST-SC-024: The sessions come 20 at a time, the latest start first, each row once,
// also when two rows share a start.
func TestTheSessionsComeTwentyAtATime(t *testing.T) {
	t.Parallel()

	rig := newHistoryRig(t)
	sessions := make([]model.Session, 45)

	for i := range sessions {
		sessions[i] = sessionAt(
			int64(i+1),
			lotA1637,
			plateA,
			day(time.March, 1, 0).Add(time.Duration(i)*time.Hour),
			0.55,
		)
	}

	sessions[11].StartedAt = sessions[10].StartedAt
	sessions[11].EndedAt = sessions[10].EndedAt
	sessions[0].Amount = 0

	rig.save(t, userA, sessions, nil)

	first := pageOf[sessionAnswer](t, rig.get(t, "/sessions"))
	second := pageOf[sessionAnswer](t, rig.get(t, follow(t, first.Next)))
	third := pageOf[sessionAnswer](t, rig.get(t, follow(t, second.Next)))
	back := pageOf[sessionAnswer](t, rig.get(t, follow(t, third.Prev)))

	assert.Len(t, first.Items, 20)
	assert.Len(t, second.Items, 20)
	assert.Len(t, third.Items, 5)
	assert.Nil(t, third.Next)
	assert.Equal(t, second.Items, back.Items)

	seen := map[string]bool{}
	all := slices.Concat(first.Items, second.Items, third.Items)

	for i, row := range all {
		assert.False(t, seen[row.ID], "row %s twice", row.ID)
		seen[row.ID] = true

		if i > 0 {
			assert.False(t, row.StartedAt.After(all[i-1].StartedAt))
		}
	}

	last := all[len(all)-1]
	assert.True(t, last.Free)
	assert.Zero(t, last.Amount.Amount)
	assert.Equal(t, sessionAnswer{
		ID: last.ID, LotNumber: lotA1637, Address: "Street of A1637", Plate: plateA,
		StartedAt: day(time.March, 1, 0), EndedAt: day(time.March, 1, 0).Add(30 * time.Minute),
		DurationSeconds: 1800, Amount: money{Currency: "GEL"}, Free: true,
	}, last)
}

// PARKHIST-SC-025: The list and the total hold every session of the plate, the dates,
// and the paid filter, not only the shown page.
func TestTheSessionFiltersNarrowTheListAndTheTotal(t *testing.T) {
	t.Parallel()

	rig := newHistoryRig(t)

	sessions := make([]model.Session, 0, 46)

	for i := range 30 {
		amount := 0.50
		if i < 8 {
			amount = 0
		}

		sessions = append(
			sessions,
			sessionAt(int64(i+1), lotA1637, plateA, day(time.November, 1+i%28, 9), amount),
		)
	}

	for i := range 15 {
		sessions = append(
			sessions,
			sessionAt(int64(100+i), lotA1637, plateB, day(time.November, 1+i, 9), 0.50),
		)
	}

	sessions = append(sessions, sessionAt(200, lotA1637, plateA, day(time.December, 1, 9), 0.50))
	rig.save(t, userA, sessions, nil)

	const filter = "plate=" + plateA + "&from=2026-11-01&to=2026-11-30&free=false"

	first := pageOf[sessionAnswer](t, rig.get(t, "/sessions?"+filter))
	second := pageOf[sessionAnswer](t, rig.get(t, follow(t, first.Next)))
	total := totalOf[sessionTotal](t, rig.get(t, "/sessions/total?"+filter))

	for _, row := range slices.Concat(first.Items, second.Items) {
		assert.Equal(t, plateA, row.Plate)
		assert.False(t, row.Free)
		assert.Equal(t, time.November, row.StartedAt.Month())
	}

	assert.Len(t, slices.Concat(first.Items, second.Items), 22)
	assert.Equal(t, 22, total.Count)
	assert.Equal(t, int64(22*1800), total.DurationSeconds)
	assert.InDelta(t, 11.00, total.Amount.Amount, 1e-9)
}

// PARKHIST-SC-026: A period in the wrong order, a date that does not exist, and a free
// that is not a boolean each answer 400, with a detail that names the parameter.
func TestABadSessionFilterIsRefused(t *testing.T) {
	t.Parallel()

	routes := historyRoutes(service.NewHistoryStore(nil))

	for target, parameter := range map[string]string{
		"/sessions?from=2026-11-30&to=2026-11-01": "from",
		"/sessions?from=2026-13-01":               "from",
		"/sessions?free=yes":                      parameterFree,
		"/sessions/total?free=yes":                parameterFree,
		"/transactions?kind=refund":               "kind",
	} {
		recorder := serve(t, routes, target)

		require.Equal(t, http.StatusBadRequest, recorder.Code, target)

		answer := decode(t, recorder)
		assert.Equal(t, "/problems/http/request-invalid", answer["type"], target)
		assert.Contains(t, answer["detail"], "The "+parameter+" parameter", target)
	}
}

// PARKHIST-SC-027: The plates are those of the user. A plate of another user answers an
// empty page.
func TestThePlatesAreThoseOfTheUser(t *testing.T) {
	t.Parallel()

	rig := newHistoryRig(t)
	rig.save(t, userA, []model.Session{
		sessionAt(1, lotA1637, plateA, day(time.March, 1, 9), 0.5),
		sessionAt(2, lotA1637, plateB, day(time.March, 2, 9), 0.5),
		sessionAt(3, lotA1637, plateA, day(time.March, 3, 9), 0.5),
	}, nil)
	rig.save(
		t,
		userB,
		[]model.Session{sessionAt(4, lotA1637, "ZZ999ZZ", day(time.March, 1, 9), 0.5)},
		nil,
	)

	type plate struct {
		Plate string `json:"plate"`
	}

	plates := pageOf[plate](t, rig.get(t, "/plates"))
	assert.Equal(t, []plate{{Plate: plateA}, {Plate: plateB}}, plates.Items)

	other := pageOf[sessionAnswer](t, rig.get(t, "/sessions?plate=ZZ999ZZ"))
	assert.Empty(t, other.Items)
}

// PARKHIST-SC-028: The transactions come 20 at a time, the latest moment first. A payment
// names the lot of a stored session, and none for an absent session.
// PARKHIST-SC-017: No answer holds the wallet number or the personal identifier.
func TestTheTransactionsNameTheLotOfTheirSession(t *testing.T) {
	t.Parallel()

	rig := newHistoryRig(t)
	rig.save(
		t,
		userA,
		[]model.Session{sessionAt(45304769, lotA1637, plateA, day(time.March, 1, 9), 0.55)},
		nil,
	)

	rig.save(t, userA, nil, ledgerOfTwentyFive())

	firstAnswer := rig.get(t, "/transactions")
	first := pageOf[transactionAnswer](t, firstAnswer)
	second := pageOf[transactionAnswer](t, rig.get(t, follow(t, first.Next)))

	require.Len(t, first.Items, 20)
	require.Len(t, second.Items, 5)

	assert.Equal(t, "permit_purchase", first.Items[0].Kind)
	assert.Equal(t, "other", first.Items[1].Kind)
	assert.Equal(t, "parking_payment", first.Items[2].Kind)
	assert.Nil(t, first.Items[2].Session)
	assert.Equal(
		t,
		&lotAnswer{LotNumber: lotA1637, Address: "Street of A1637"},
		first.Items[3].Session,
	)
	assert.InDelta(t, 0.55, first.Items[3].Amount.Amount, 1e-9)
	assert.InDelta(t, 9.45, first.Items[3].BalanceAfter.Amount, 1e-9)

	for _, row := range slices.Concat(first.Items, second.Items) {
		assert.GreaterOrEqual(t, row.Amount.Amount, 0.0)
	}

	assert.NotContains(t, firstAnswer.Body.String(), "wallet")
	assert.NotContains(t, firstAnswer.Body.String(), "person")
}

// ledgerOfTwentyFive answers 21 top ups, a payment of the stored session 45304769, a
// payment of an absent session, a transaction of another kind, and a permit purchase.
func ledgerOfTwentyFive() []model.Transaction {
	transactions := make([]model.Transaction, 0, 25)
	for i := range 21 {
		transactions = append(
			transactions,
			transactionAt(
				int64(100+i),
				model.TransactionKindTopUp,
				day(time.February, 1+i, 9),
				10,
				10,
			),
		)
	}

	stored, absent := int64(45304769), int64(99)
	paid := transactionAt(
		1,
		model.TransactionKindParkingPayment,
		day(time.March, 1, 10),
		0.55,
		9.45,
	)
	paid.SessionExternalID = &stored
	lost := transactionAt(
		2,
		model.TransactionKindParkingPayment,
		day(time.March, 2, 10),
		0.50,
		8.95,
	)
	lost.SessionExternalID = &absent
	other := transactionAt(3, model.TransactionKindOther, day(time.March, 3, 10), 2, 10.95)
	permit := transactionAt(4, model.TransactionKindPermitPurchase, day(time.March, 4, 10), 5, 5.95)

	return append(transactions, paid, lost, other, permit)
}

// PARKHIST-SC-029: The total of a month sums the top ups and the charges, and the kind
// narrows the list.
func TestTheTransactionTotalSumsTopUpsAndCharges(t *testing.T) {
	t.Parallel()

	rig := newHistoryRig(t)

	var transactions []model.Transaction

	id := int64(1)
	add := func(kind model.TransactionKind, amount float64, date int) {
		transactions = append(
			transactions,
			transactionAt(id, kind, day(time.October, date, 9), amount, 1),
		)
		id++
	}

	for date := 1; date <= 3; date++ {
		add(model.TransactionKindTopUp, 10, date)
	}

	for date := 4; date <= 15; date++ {
		add(model.TransactionKindParkingPayment, 0.50, date)
	}

	add(model.TransactionKindPermitPurchase, 25, 20)

	transactions = append(transactions,
		transactionAt(99, model.TransactionKindTopUp, day(time.November, 1, 9), 50, 1))
	rig.save(t, userA, nil, transactions)

	total := totalOf[transactionTotal](
		t,
		rig.get(t, "/transactions/total?from=2026-10-01&to=2026-10-31"),
	)
	assert.Equal(t, 16, total.Count)
	assert.InDelta(t, 30.00, total.TopUps.Amount, 1e-9)
	assert.InDelta(t, 31.00, total.Charges.Amount, 1e-9)

	permits := pageOf[transactionAnswer](t, rig.get(t, "/transactions?kind=permit_purchase"))
	require.Len(t, permits.Items, 1)
	assert.InDelta(t, 25.00, permits.Items[0].Amount.Amount, 1e-9)
}
