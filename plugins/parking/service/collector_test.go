package service_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/parking/dto"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/model"
	"github.com/abgeo/maroid/plugins/parking/service"
)

const (
	pageSize = 20
	endless  = -1

	listTransactions = "transactions"
	firstSession     = 1000
	firstEntry       = 5000

	startDate = "2026-11-21T13:26:08.347292"
	endDate   = "2026-11-21T14:14:21.809195"
	moment    = "2026-11-21T10:14:21.827+0000"
)

// fakeParking answers the two lists of the parking service from memory, and records each
// page that a call reads. A total of endless answers full pages with no end.
type fakeParking struct {
	mu           sync.Mutex
	sessions     []dto.HistorySession
	transactions []dto.PaymentEntry
	sessionTotal int
	failAt       map[string]int
	sessionPages []int
	ledgerPages  []int
}

func (f *fakeParking) ParkingHistory(
	_ context.Context,
	page int,
) (dto.Page[dto.HistorySession], error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sessionPages = append(f.sessionPages, page)

	if f.sessionTotal == endless {
		rows := make([]dto.HistorySession, pageSize)
		for i := range rows {
			rows[i] = finished(int64(page*pageSize + i + 1))
		}

		return dto.Page[dto.HistorySession]{
			Items: rows,
			Meta:  dto.PageMeta{Total: (page + 2) * pageSize, Number: page, Size: pageSize},
		}, nil
	}

	return pageOf(f.sessions, page), nil
}

func (f *fakeParking) PaymentHistory(
	_ context.Context,
	page int,
) (dto.Page[dto.PaymentEntry], error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.ledgerPages = append(f.ledgerPages, page)

	if failing, found := f.failAt[listTransactions]; found && failing == page {
		return dto.Page[dto.PaymentEntry]{}, &service.StatusError{Status: 500}
	}

	return pageOf(f.transactions, page), nil
}

func pageOf[T any](rows []T, page int) dto.Page[T] {
	start := min(page*pageSize, len(rows))
	end := min(start+pageSize, len(rows))

	return dto.Page[T]{
		Items: rows[start:end],
		Meta:  dto.PageMeta{Total: len(rows), Number: page, Size: pageSize},
	}
}

func finished(id int64) dto.HistorySession {
	end := endDate

	return dto.HistorySession{
		ID: id, Status: "FINISHED", Type: "BOTH", GovNumber: "AB071EO",
		ParkingPlace: dto.HistoryPlace{UniqueNumber: "A1637", Address: "12 Example Street"},
		StartDate:    startDate, EndDate: &end, Amount: 0.55,
	}
}

func entry(id int64, date string) dto.PaymentEntry {
	return dto.PaymentEntry{
		ID: id, OperationType: "ZONE_PARKING_PAYMENT", Amount: 0.55, CurrentBalance: 9.08,
		Description: "Zone Parking", OperationDate: date,
	}
}

// sessionsOf answers count finished sessions, the newest first, from firstSession down.
func sessionsOf(count int) []dto.HistorySession {
	rows := make([]dto.HistorySession, count)
	for i := range rows {
		rows[i] = finished(firstSession - int64(i))
	}

	return rows
}

// entriesOf answers count transactions, the newest first, from firstEntry down.
func entriesOf(count int) []dto.PaymentEntry {
	rows := make([]dto.PaymentEntry, count)
	for i := range rows {
		rows[i] = entry(firstEntry-int64(i), moment)
	}

	return rows
}

// fakeStore holds external identifiers in memory, and records each save.
type fakeStore struct {
	mu           sync.Mutex
	sessions     map[int64]bool
	transactions map[int64]bool
	saves        int
	saved        model.Collection
	stored       []model.Session
	ledger       []model.Transaction
}

func newFakeStore() *fakeStore {
	return &fakeStore{sessions: map[int64]bool{}, transactions: map[int64]bool{}}
}

func (f *fakeStore) KnownSessions(_ context.Context, ids []int64) (map[int64]bool, error) {
	return f.known(f.sessions, ids), nil
}

func (f *fakeStore) KnownTransactions(_ context.Context, ids []int64) (map[int64]bool, error) {
	return f.known(f.transactions, ids), nil
}

func (f *fakeStore) Save(
	_ context.Context,
	sessions []model.Session,
	transactions []model.Transaction,
) (model.Collection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.saves++
	f.stored = append(f.stored, sessions...)
	f.ledger = append(f.ledger, transactions...)

	var collection model.Collection

	for _, session := range sessions {
		if !f.sessions[session.ExternalID] {
			f.sessions[session.ExternalID] = true
			collection.Sessions++
		}
	}

	for _, transaction := range transactions {
		if !f.transactions[transaction.ExternalID] {
			f.transactions[transaction.ExternalID] = true
			collection.Transactions++
		}
	}

	f.saved = collection

	return collection, nil
}

func (f *fakeStore) known(held map[int64]bool, ids []int64) map[int64]bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	known := map[int64]bool{}

	for _, id := range ids {
		if held[id] {
			known[id] = true
		}
	}

	return known
}

func (f *fakeStore) hold(sessions []dto.HistorySession, transactions []dto.PaymentEntry) {
	for _, session := range sessions {
		f.sessions[session.ID] = true
	}

	for _, transaction := range transactions {
		f.transactions[transaction.ID] = true
	}
}

func collectorOf(parking *fakeParking, store *fakeStore) (*service.Collector, *bytes.Buffer) {
	var log bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&log, nil))

	return service.NewCollector(parking, store, logger), &log
}

// PARKHIST-SC-004: A first page of held rows ends each list, and nothing is written.
func TestAQuietHourReadsOnePageOfEachList(t *testing.T) {
	t.Parallel()

	parking := &fakeParking{sessions: sessionsOf(60), transactions: entriesOf(60)}
	store := newFakeStore()
	store.hold(parking.sessions[:pageSize], parking.transactions[:pageSize])

	collector, _ := collectorOf(parking, store)

	collection, err := collector.Collect(as(userA), model.ReachRecent(5))
	require.NoError(t, err)

	assert.Equal(t, []int{0}, parking.sessionPages)
	assert.Equal(t, []int{0}, parking.ledgerPages)
	assert.Zero(t, store.saves)
	assert.Equal(t, model.Collection{}, collection)
}

// PARKHIST-SC-005: A page with one new row reads the next page. A new transaction dated
// before a held row of the same page is stored.
func TestANewRowReadsTheNextPage(t *testing.T) {
	t.Parallel()

	parking := &fakeParking{sessions: sessionsOf(60), transactions: entriesOf(60)}
	parking.transactions[1] = entry(4000, "2026-11-20T08:00:00.000+0000")

	store := newFakeStore()
	store.hold(parking.sessions[1:], nil)
	store.hold(nil, parking.transactions[:1])
	store.hold(nil, parking.transactions[2:])

	collector, _ := collectorOf(parking, store)

	collection, err := collector.Collect(as(userA), model.ReachRecent(5))
	require.NoError(t, err)

	assert.Equal(t, []int{0, 1}, parking.sessionPages)
	assert.Equal(t, []int{0, 1}, parking.ledgerPages)
	assert.Equal(t, model.Collection{Sessions: 1, Transactions: 1}, collection)
	require.Len(t, store.stored, 1)
	assert.Equal(t, int64(1000), store.stored[0].ExternalID)
	require.Len(t, store.ledger, 1)
	assert.Equal(t, int64(4000), store.ledger[0].ExternalID)
}

// PARKHIST-SC-006: The recent reach stops at the page limit, and warns for the list that
// still held a new row.
func TestTheRecentReachStopsAtThePageLimit(t *testing.T) {
	t.Parallel()

	parking := &fakeParking{sessions: sessionsOf(468), transactions: entriesOf(55)}
	store := newFakeStore()

	collector, log := collectorOf(parking, store)

	collection, err := collector.Collect(as(userA), model.ReachRecent(5))
	require.NoError(t, err)

	assert.Equal(t, []int{0, 1, 2, 3, 4}, parking.sessionPages)
	assert.Equal(t, []int{0, 1, 2}, parking.ledgerPages)
	assert.Equal(t, model.Collection{Sessions: 100, Transactions: 55}, collection)

	warnings := strings.Count(log.String(), "level=WARN")
	assert.Equal(t, 1, warnings)
	assert.Contains(t, log.String(), "list=sessions")
	assert.Contains(t, log.String(), "run backfill")
}

// PARKHIST-SC-007: An active session is not stored, and counts as a held row.
func TestAnActiveSessionIsNotStored(t *testing.T) {
	t.Parallel()

	parking := &fakeParking{sessions: sessionsOf(60), transactions: entriesOf(20)}
	parking.sessions[0] = dto.HistorySession{ID: 1001, Status: "ACTIVE", StartDate: startDate}

	store := newFakeStore()
	store.hold(parking.sessions[1:], parking.transactions)

	collector, _ := collectorOf(parking, store)

	_, err := collector.Collect(as(userA), model.ReachRecent(5))
	require.NoError(t, err)

	assert.Equal(t, []int{0}, parking.sessionPages)
	assert.Empty(t, store.stored)
}

// PARKHIST-SC-018: A full reach of a list with no end fails at the page limit, and
// writes nothing.
func TestAListWithNoEndFailsTheFullReach(t *testing.T) {
	t.Parallel()

	parking := &fakeParking{sessionTotal: endless}
	store := newFakeStore()

	collector, _ := collectorOf(parking, store)

	_, err := collector.Collect(as(userA), model.ReachFull())

	require.ErrorIs(t, err, errs.ErrPageLimit)
	assert.Len(t, parking.sessionPages, 500)
	assert.Equal(t, 499, parking.sessionPages[len(parking.sessionPages)-1])
	assert.Empty(t, parking.ledgerPages)
	assert.Zero(t, store.saves)
}

// PARKHIST-SC-014: The full reach reads every page of both lists, saves every row, and
// counts the new ones.
func TestTheFullReachReadsEveryPage(t *testing.T) {
	t.Parallel()

	parking := &fakeParking{sessions: sessionsOf(468), transactions: entriesOf(393)}
	store := newFakeStore()
	store.hold(parking.sessions[100:130], nil)

	collector, _ := collectorOf(parking, store)

	collection, err := collector.Collect(as(userA), model.ReachFull())
	require.NoError(t, err)

	assert.Len(t, parking.sessionPages, 24)
	assert.Len(t, parking.ledgerPages, 20)
	assert.Len(t, store.stored, 468)
	assert.Len(t, store.ledger, 393)
	assert.Equal(t, model.Collection{Sessions: 438, Transactions: 393}, collection)
}

// PARKHIST-SC-008: A failed page fails the collection, and nothing is written.
func TestAFailedPageWritesNothing(t *testing.T) {
	t.Parallel()

	parking := &fakeParking{
		sessions:     sessionsOf(10),
		transactions: entriesOf(60),
		failAt:       map[string]int{listTransactions: 2},
	}
	store := newFakeStore()

	collector, _ := collectorOf(parking, store)

	_, err := collector.Collect(as(userA), model.ReachFull())

	require.ErrorIs(t, err, errs.ErrRequestFailed)
	assert.Zero(t, store.saves)
}
