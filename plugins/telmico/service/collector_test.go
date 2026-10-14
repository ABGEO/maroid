package service_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/dto"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
	"github.com/abgeo/maroid/plugins/telmico/service"
)

const (
	secretSession = "session"
	cookieK0      = "COOKIE-K0"
)

//nolint:gochecknoglobals
var september = model.Month{Year: 2026, Month: time.September}

// memorySecrets is a SecretProvider in memory, keyed by the acting workspace and the name.
type memorySecrets struct {
	mu      sync.Mutex
	entries map[string]string
}

func secretsHolding(session *model.Session) *memorySecrets {
	secrets := &memorySecrets{entries: map[string]string{}}

	if session != nil {
		stored, _ := session.MarshalSecret()
		secrets.entries[workspaceW+"/"+secretSession] = stored
	}

	return secrets
}

func (m *memorySecrets) ReadSecret(
	ctx context.Context,
	_ *pluginapi.PluginID,
	_ pluginapi.SecretScope,
	name string,
) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, found := m.entries[pluginapi.ActingWorkspaceFromContext(ctx)+"/"+name]
	if !found {
		return "", pluginapi.ErrSecretAbsent
	}

	return value, nil
}

func (m *memorySecrets) WriteSecret(
	ctx context.Context,
	_ *pluginapi.PluginID,
	_ pluginapi.SecretScope,
	name string,
	value string,
	_ time.Time,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries[pluginapi.ActingWorkspaceFromContext(ctx)+"/"+name] = value

	return nil
}

func (m *memorySecrets) DeleteSecret(
	ctx context.Context,
	_ *pluginapi.PluginID,
	_ pluginapi.SecretScope,
	name string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.entries, pluginapi.ActingWorkspaceFromContext(ctx)+"/"+name)

	return nil
}

// session answers the session of workspace W, and whether one exists.
func (m *memorySecrets) session(t *testing.T) (model.Session, bool) {
	t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	stored, found := m.entries[workspaceW+"/"+secretSession]
	if !found {
		return model.Session{}, false
	}

	session, err := model.ParseSession(stored)
	require.NoError(t, err)

	return session, true
}

// storedCredentials answers the username and the password of the workspace.
type storedCredentials struct{}

func (storedCredentials) Settings(context.Context, *pluginapi.PluginID) (map[string]any, error) {
	return map[string]any{"username": username, "password": password}, nil
}

// memoryStore holds the assignment, the transactions, and the bills of workspace W in
// memory. It keeps one transaction for one kind, date, and meter value, as the table does.
type memoryStore struct {
	mu           sync.Mutex
	assigned     string
	transactions map[string]model.Transaction
	bills        map[string]model.Bill
}

func storeAssigning(customer string) *memoryStore {
	return &memoryStore{
		assigned:     customer,
		transactions: map[string]model.Transaction{},
		bills:        map[string]model.Bill{},
	}
}

func transactionKey(transaction model.Transaction) string {
	meter := "none"
	if transaction.MeterValue != nil {
		meter = time.Duration(*transaction.MeterValue * 1000).String()
	}

	return transaction.CustomerNumber + "/" + transaction.Operation + "/" +
		transaction.Date.Format(time.DateOnly) + "/" + meter
}

func (m *memoryStore) AssignedCustomer(context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.assigned == "" {
		return "", errs.ErrNotAssigned
	}

	return m.assigned, nil
}

func (m *memoryStore) InsertTransactions(
	_ context.Context,
	transactions []model.Transaction,
) ([]model.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var stored []model.Transaction

	for _, transaction := range transactions {
		key := transactionKey(transaction)
		if _, found := m.transactions[key]; !found {
			stored = append(stored, transaction)
		}

		m.transactions[key] = transaction
	}

	return stored, nil
}

func (m *memoryStore) InsertBills(_ context.Context, bills []model.Bill) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, bill := range bills {
		m.bills[bill.BillNumber] = bill
	}

	return nil
}

func (m *memoryStore) ChargesOn(
	_ context.Context,
	customer string,
	day time.Time,
) ([]model.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var charges []model.Transaction

	for _, transaction := range m.transactions {
		if transaction.CustomerNumber == customer && transaction.IsCharge() &&
			transaction.Date.Format(time.DateOnly) == day.Format(time.DateOnly) {
			charges = append(charges, transaction)
		}
	}

	return charges, nil
}

func (m *memoryStore) counts() (int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.transactions), len(m.bills)
}

// fakeClient plays the portal at the level of the client. It accepts the cookies that
// it holds, and records each call.
type fakeClient struct {
	mu        sync.Mutex
	accepted  []string
	signIn    func() (model.Session, error)
	accounts  []dto.Account
	payments  []dto.Payment
	receipts  []dto.Receipt
	calls     []string
	months    []string
	signInRun int
}

func portalAccepting(cookies ...string) *fakeClient {
	return &fakeClient{
		accepted: cookies,
		signIn: func() (model.Session, error) {
			return model.Session{Cookie: cookieK, CSRFToken: csrfT}, nil
		},
		accounts: []dto.Account{{Number: customerC1, Name: "House"}},
	}
}

func (f *fakeClient) SignIn(context.Context, string, string) (model.Session, error) {
	f.mu.Lock()
	f.calls = append(f.calls, "sign in")
	f.signInRun++
	signIn := f.signIn
	f.mu.Unlock()

	return signIn()
}

func (f *fakeClient) Accounts(_ context.Context, session model.Session) ([]dto.Account, error) {
	if err := f.record("accounts", session); err != nil {
		return nil, err
	}

	return f.accounts, nil
}

func (f *fakeClient) AccountDetail(
	_ context.Context,
	session model.Session,
	account string,
) (dto.AccountDetail, error) {
	if err := f.record("detail", session); err != nil {
		return dto.AccountDetail{}, err
	}

	return dto.AccountDetail{Number: account, Address: "Nutsubidze 4"}, nil
}

func (f *fakeClient) Payments(
	_ context.Context,
	session model.Session,
	_ string,
	month model.Month,
) ([]dto.Payment, error) {
	if err := f.record("payments", session); err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.months = append(f.months, month.PortalForm())
	f.mu.Unlock()

	return f.payments, nil
}

func (f *fakeClient) Receipts(
	_ context.Context,
	session model.Session,
	_ string,
	month model.Month,
) ([]dto.Receipt, error) {
	if err := f.record("receipts", session); err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.months = append(f.months, month.PortalForm())
	f.mu.Unlock()

	return f.receipts, nil
}

func (f *fakeClient) ReceiptDocument(
	_ context.Context,
	session model.Session,
	_ uuid.UUID,
) ([]byte, error) {
	if err := f.record("document", session); err != nil {
		return nil, err
	}

	return []byte("%PDF-1.3"), nil
}

func (f *fakeClient) record(name string, session model.Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, name)

	if !slices.Contains(f.accepted, session.Cookie) {
		return errs.ErrSessionRejected
	}

	return nil
}

func (f *fakeClient) recorded() ([]string, []string, int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.calls), slices.Clone(f.months), f.signInRun
}

func septemberAnswers(portal *fakeClient) *fakeClient {
	portal.payments = []dto.Payment{
		{
			OperationDate: "2026-09-28", TransactionType: operationCharge,
			CounterValue: new(1989.0), KiloValue: new(162.6), Amount: new(39.11),
			Debt: new(39.11), CounterNumber: new("021242007390"), TariffValue: new(0.2405312),
		},
		{
			OperationDate:   "2026-09-01",
			TransactionType: operationPayment,
			Amount:          new(-18.92),
			Debt:            new(0.0),
		},
	}
	portal.receipts = []dto.Receipt{{
		BillID:         "f9ca32a7-ec95-4145-97af-cbef330af8dc",
		BillNumber:     "8370764-2026-9",
		BillCreateDate: "2026-09-30T09:29:45.299",
	}}

	return portal
}

func collectorOf(
	portal *fakeClient,
	secrets *memorySecrets,
	store *memoryStore,
) *service.Collector {
	identifier := pluginapi.ParsePluginID(pluginID)

	return service.NewCollector(
		portal,
		pluginapi.NewPluginSettings(storedCredentials{}, identifier),
		pluginapi.NewPluginSecrets(secrets, identifier),
		store,
	)
}

func actingInW(t *testing.T) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingWorkspace(t.Context(), workspaceW)
}

// ELECACC-SC-012: A collection of September reads the month in the form of the portal,
// stores the transactions and the bill, and names the new reading. A repeat names none.
func TestTheCollectionStoresTheMonthAndNamesTheNewReading(t *testing.T) {
	t.Parallel()

	portal := septemberAnswers(portalAccepting(cookieK))
	store := storeAssigning(customerC1)
	collector := collectorOf(portal, secretsHolding(new(session())), store)

	collection, err := collector.Collect(actingInW(t), september)
	require.NoError(t, err)

	assert.Equal(t, 2, collection.Count)
	require.NotNil(t, collection.NewestReading)
	assert.Equal(t, "2026-09-28", collection.NewestReading.Date.Format(time.DateOnly))
	assert.InDelta(t, 162.6, collection.NewestReading.Usage, 1e-9)
	assert.InDelta(t, 39.11, collection.NewestReading.Charge, 1e-9)

	transactions, bills := store.counts()
	assert.Equal(t, 2, transactions)
	assert.Equal(t, 1, bills)

	_, months, _ := portal.recorded()
	assert.Equal(t, []string{septemberForm, septemberForm}, months)

	repeat, err := collector.Collect(actingInW(t), september)
	require.NoError(t, err)
	assert.Nil(t, repeat.NewestReading)
}

// ELECACC-SC-021: The two charges of a split reading form the new reading, also when one
// of them was stored before.
func TestTheNewReadingHoldsEveryChargeOfItsDate(t *testing.T) {
	t.Parallel()

	low := dto.Payment{
		OperationDate: "2026-04-28", TransactionType: operationCharge,
		CounterValue: new(1163.0), KiloValue: new(13.2), Amount: new(2.52), Debt: new(39.39),
	}
	high := dto.Payment{
		OperationDate: "2026-04-28", TransactionType: operationCharge,
		CounterValue: new(1317.0), KiloValue: new(153.3), Amount: new(36.87), Debt: new(39.39),
	}

	portal := portalAccepting(cookieK)
	portal.payments = []dto.Payment{low}

	store := storeAssigning(customerC1)
	collector := collectorOf(portal, secretsHolding(new(session())), store)

	_, err := collector.Collect(actingInW(t), model.Month{Year: 2026, Month: time.April})
	require.NoError(t, err)

	portal.payments = []dto.Payment{low, high}

	collection, err := collector.Collect(actingInW(t), model.Month{Year: 2026, Month: time.April})
	require.NoError(t, err)

	require.NotNil(t, collection.NewestReading)
	assert.InDelta(t, 166.5, collection.NewestReading.Usage, 1e-9)
	assert.InDelta(t, 39.39, collection.NewestReading.Charge, 1e-9)
	assert.InDelta(t, 1317.0, collection.NewestReading.MeterValue, 0)
}

// ELECACC-SC-014: A workspace with no assignment or no session is not set up, and the
// collection calls no portal.
func TestTheCollectionNamesAWorkspaceThatIsNotSetUp(t *testing.T) {
	t.Parallel()

	portal := portalAccepting(cookieK)

	_, err := collectorOf(portal, secretsHolding(new(session())), storeAssigning("")).
		Collect(actingInW(t), september)
	require.ErrorIs(t, err, errs.ErrNotAssigned)

	_, err = collectorOf(portal, secretsHolding(nil), storeAssigning(customerC1)).
		Collect(actingInW(t), september)
	require.ErrorIs(t, err, errs.ErrNotConnected)

	calls, _, _ := portal.recorded()
	assert.Empty(t, calls)
}

// ELECACC-SC-015: An account that no longer holds the assigned customer fails the
// collection, which stores nothing.
func TestTheCollectionFailsWhenTheCustomerIsAbsent(t *testing.T) {
	t.Parallel()

	portal := septemberAnswers(portalAccepting(cookieK))
	portal.accounts = []dto.Account{{Number: customerC2}}
	store := storeAssigning(customerC1)

	_, err := collectorOf(portal, secretsHolding(new(session())), store).
		Collect(actingInW(t), september)

	require.ErrorIs(t, err, errs.ErrCustomerAbsent)

	transactions, bills := store.counts()
	assert.Zero(t, transactions)
	assert.Zero(t, bills)
}

// ELECACC-SC-008: A rejected session gives way to one new sign in, whose session the
// secret keeps. A new session that the portal rejects too fails the collection.
func TestTheCollectionSignsInAgainOnce(t *testing.T) {
	t.Parallel()

	portal := septemberAnswers(portalAccepting(cookieK))
	secrets := secretsHolding(&model.Session{Cookie: cookieK0, CSRFToken: csrfT})

	_, err := collectorOf(portal, secrets, storeAssigning(customerC1)).
		Collect(actingInW(t), september)
	require.NoError(t, err)

	_, _, signIns := portal.recorded()
	assert.Equal(t, 1, signIns)

	held, found := secrets.session(t)
	require.True(t, found)
	assert.Equal(t, cookieK, held.Cookie)

	rejecting := septemberAnswers(portalAccepting())
	secrets = secretsHolding(&model.Session{Cookie: cookieK0, CSRFToken: csrfT})

	_, err = collectorOf(rejecting, secrets, storeAssigning(customerC1)).
		Collect(actingInW(t), september)
	require.ErrorIs(t, err, errs.ErrRequestFailed)

	_, _, signIns = rejecting.recorded()
	assert.Equal(t, 1, signIns)
}

// ELECACC-SC-009: Credentials that the portal rejects at the new sign in fail the
// collection with the rejection, and remove the session.
func TestTheCollectionFailsWhenTheCredentialsAreRejected(t *testing.T) {
	t.Parallel()

	portal := septemberAnswers(portalAccepting(cookieK))
	portal.signIn = func() (model.Session, error) {
		return model.Session{}, errs.ErrCredentialsRejected
	}

	secrets := secretsHolding(&model.Session{Cookie: cookieK0, CSRFToken: csrfT})
	store := storeAssigning(customerC1)

	_, err := collectorOf(portal, secrets, store).Collect(actingInW(t), september)

	require.ErrorIs(t, err, errs.ErrCredentialsRejected)

	_, found := secrets.session(t)
	assert.False(t, found)

	assigned, err := store.AssignedCustomer(t.Context())
	require.NoError(t, err)
	assert.Equal(t, customerC1, assigned)
}
