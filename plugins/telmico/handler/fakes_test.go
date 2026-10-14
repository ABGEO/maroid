package handler_test

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/dto"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

const (
	workspaceW = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e78"
	username   = "01001010101"
	password   = "p4ss-word"

	secretSession = "session"
	cookieK0      = "COOKIE-OF-AN-ENDED-SESSION"
	cookieK       = "COOKIE-OF-THE-NEW-SESSION"
	csrfT         = "CSRF-OF-THE-SESSION"

	customerC1 = "8370764"
	customerC2 = "8370999"
)

// fakeSecrets is a SecretProvider in memory, keyed by the acting workspace and the name.
type fakeSecrets struct {
	mu      sync.Mutex
	entries map[string]string
}

func newFakeSecrets(cookie string) *fakeSecrets {
	secrets := &fakeSecrets{entries: map[string]string{}}

	if cookie != "" {
		stored, _ := model.Session{Cookie: cookie, CSRFToken: csrfT}.MarshalSecret()
		secrets.entries[workspaceW+"/"+secretSession] = stored
	}

	return secrets
}

func secretKey(ctx context.Context, name string) string {
	return pluginapi.ActingWorkspaceFromContext(ctx) + "/" + name
}

func (f *fakeSecrets) ReadSecret(
	ctx context.Context,
	_ *pluginapi.PluginID,
	_ pluginapi.SecretScope,
	name string,
) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	value, found := f.entries[secretKey(ctx, name)]
	if !found {
		return "", pluginapi.ErrSecretAbsent
	}

	return value, nil
}

func (f *fakeSecrets) WriteSecret(
	ctx context.Context,
	_ *pluginapi.PluginID,
	_ pluginapi.SecretScope,
	name string,
	value string,
	_ time.Time,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.entries[secretKey(ctx, name)] = value

	return nil
}

func (f *fakeSecrets) DeleteSecret(
	ctx context.Context,
	_ *pluginapi.PluginID,
	_ pluginapi.SecretScope,
	name string,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.entries, secretKey(ctx, name))

	return nil
}

// cookie answers the cookie of the session of workspace W, empty when none is stored.
func (f *fakeSecrets) cookie() string {
	f.mu.Lock()
	defer f.mu.Unlock()

	stored, found := f.entries[workspaceW+"/"+secretSession]
	if !found {
		return ""
	}

	session, err := model.ParseSession(stored)
	if err != nil {
		return ""
	}

	return session.Cookie
}

// fakeSettings answers the stored credentials, or ErrSettingsAbsent when it holds none.
type fakeSettings struct {
	values map[string]any
}

func (f *fakeSettings) Settings(context.Context, *pluginapi.PluginID) (map[string]any, error) {
	if f.values == nil {
		return nil, pluginapi.ErrSettingsAbsent
	}

	return f.values, nil
}

func storedCredentials() *fakeSettings {
	return &fakeSettings{values: map[string]any{"username": username, "password": password}}
}

// fakeStore holds the assignment, the transactions, and the bills of workspace W in
// memory, as the tables do.
type fakeStore struct {
	mu           sync.Mutex
	assigned     string
	transactions map[string]model.Transaction
	bills        []model.Bill
}

func newFakeStore(assigned string) *fakeStore {
	return &fakeStore{assigned: assigned, transactions: map[string]model.Transaction{}}
}

func (f *fakeStore) AssignedCustomer(context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.assigned == "" {
		return "", errs.ErrNotAssigned
	}

	return f.assigned, nil
}

func (f *fakeStore) Assign(_ context.Context, customerNumber string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.assigned = customerNumber

	return nil
}

func (f *fakeStore) Unassign(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.assigned = ""

	return nil
}

func (f *fakeStore) InsertTransactions(
	_ context.Context,
	transactions []model.Transaction,
) ([]model.Transaction, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var stored []model.Transaction

	for _, transaction := range transactions {
		key := transaction.Operation + "/" + transaction.Date.Format(time.DateOnly)
		if _, found := f.transactions[key]; !found {
			stored = append(stored, transaction)
		}

		f.transactions[key] = transaction
	}

	return stored, nil
}

func (f *fakeStore) InsertBills(_ context.Context, bills []model.Bill) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, bill := range bills {
		if !slices.ContainsFunc(
			f.bills,
			func(held model.Bill) bool { return held.BillNumber == bill.BillNumber },
		) {
			bill.ID = uuid.NewString()
			f.bills = append(f.bills, bill)
		}
	}

	return nil
}

func (f *fakeStore) ChargesOn(
	_ context.Context,
	customer string,
	day time.Time,
) ([]model.Transaction, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var charges []model.Transaction

	for _, transaction := range f.transactions {
		if transaction.CustomerNumber == customer && transaction.IsCharge() &&
			transaction.Date.Format(time.DateOnly) == day.Format(time.DateOnly) {
			charges = append(charges, transaction)
		}
	}

	return charges, nil
}

func (f *fakeStore) BillByMonth(
	_ context.Context,
	customer string,
	month model.Month,
) (*model.Bill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, bill := range f.bills {
		if bill.CustomerNumber == customer && model.MonthOf(bill.Month) == month {
			return &bill, nil
		}
	}

	return nil, nil //nolint:nilnil // no bill is no failure
}

func (f *fakeStore) Bill(_ context.Context, id string) (model.Bill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.assigned == "" {
		return model.Bill{}, errs.ErrNotAssigned
	}

	for _, bill := range f.bills {
		if bill.ID == id && bill.CustomerNumber == f.assigned {
			return bill, nil
		}
	}

	return model.Bill{}, errs.ErrBillUnknown
}

func (f *fakeStore) state() (string, int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.assigned, len(f.transactions), len(f.bills)
}

// fakeSender records each notification.
type fakeSender struct {
	mu       sync.Mutex
	messages []pluginapi.Message
}

func (f *fakeSender) Send(
	_ context.Context,
	_ *pluginapi.PluginID,
	_ string,
	msg pluginapi.Message,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.messages = append(f.messages, msg)

	return nil
}

func (f *fakeSender) sent() []pluginapi.Message {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]pluginapi.Message(nil), f.messages...)
}

// fakePortal plays the portal at the level of the client. It accepts the cookies that
// it holds, answers each call with what the test sets, and records each call.
type fakePortal struct {
	mu       sync.Mutex
	accepted []string
	signIn   error
	// signIns answers the sign ins one at a time, a nil entry a success. signIn answers the rest.
	signIns   []error
	failure   error
	accounts  []dto.Account
	details   map[string]dto.AccountDetail
	payments  []dto.Payment
	receipts  []dto.Receipt
	document  []byte
	calls     []string
	months    []string
	documents []uuid.UUID
}

// portalWithTwoCustomers accepts cookie K, signs in with cookie K, and lists C1 and C2.
func portalWithTwoCustomers() *fakePortal {
	return &fakePortal{
		accepted: []string{cookieK},
		accounts: []dto.Account{
			{Number: customerC1, Name: " House "},
			{Number: customerC2, Name: "Garden"},
		},
		details: map[string]dto.AccountDetail{
			customerC1: {Number: customerC1, Address: " Nutsubidze 4 "},
			customerC2: {Number: customerC2, Address: "Gldani 3"},
		},
		document: []byte("%PDF-1.3 september"),
	}
}

func (f *fakePortal) SignIn(_ context.Context, user string, secret string) (model.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, "sign in")

	if len(f.signIns) > 0 {
		answer := f.signIns[0]
		f.signIns = f.signIns[1:]

		if answer != nil {
			return model.Session{}, answer
		}

		return model.Session{Cookie: cookieK, CSRFToken: csrfT}, nil
	}

	if f.signIn != nil {
		return model.Session{}, f.signIn
	}

	if user != username || secret != password {
		return model.Session{}, errs.ErrCredentialsRejected
	}

	return model.Session{Cookie: cookieK, CSRFToken: csrfT}, nil
}

func (f *fakePortal) Accounts(_ context.Context, session model.Session) ([]dto.Account, error) {
	if err := f.enter("accounts", session); err != nil {
		return nil, err
	}

	return f.accounts, nil
}

func (f *fakePortal) AccountDetail(
	_ context.Context,
	session model.Session,
	account string,
) (dto.AccountDetail, error) {
	if err := f.enter("detail", session); err != nil {
		return dto.AccountDetail{}, err
	}

	return f.details[account], nil
}

func (f *fakePortal) Payments(
	_ context.Context,
	session model.Session,
	_ string,
	month model.Month,
) ([]dto.Payment, error) {
	if err := f.enter("payments", session); err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.months = append(f.months, month.PortalForm())
	f.mu.Unlock()

	return f.payments, nil
}

func (f *fakePortal) Receipts(
	_ context.Context,
	session model.Session,
	_ string,
	month model.Month,
) ([]dto.Receipt, error) {
	if err := f.enter("receipts", session); err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.months = append(f.months, month.PortalForm())
	f.mu.Unlock()

	return f.receipts, nil
}

func (f *fakePortal) ReceiptDocument(
	_ context.Context,
	session model.Session,
	billID uuid.UUID,
) ([]byte, error) {
	if err := f.enter("document", session); err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.documents = append(f.documents, billID)
	f.mu.Unlock()

	return f.document, nil
}

func (f *fakePortal) enter(name string, session model.Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, name)

	if !slices.Contains(f.accepted, session.Cookie) {
		return errs.ErrSessionRejected
	}

	return f.failure
}

func (f *fakePortal) recorded() ([]string, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.calls), slices.Clone(f.months)
}

func (f *fakePortal) countOf(name string) int {
	calls, _ := f.recorded()

	count := 0

	for _, recorded := range calls {
		if recorded == name {
			count++
		}
	}

	return count
}

func chargeOf(customer string) model.Transaction {
	return model.Transaction{
		CustomerNumber: customer,
		Operation:      "დარიცხვა ჩვენების მიხედვით",
		Date:           time.Date(2026, time.September, 28, 0, 0, 0, 0, model.Tbilisi()),
		MeterValue:     new(1989.0),
		Usage:          new(162.6),
		Amount:         new(39.11),
		Balance:        new(39.11),
		Tariff:         new(0.2405312),
	}
}

func septemberBill(customer string) model.Bill {
	return model.Bill{
		ID:             uuid.NewString(),
		CustomerNumber: customer,
		BillNumber:     customer + "-2026-9",
		ServiceBillID:  uuid.MustParse("f9ca32a7-ec95-4145-97af-cbef330af8dc"),
		Month:          model.Month{Year: 2026, Month: time.September}.First(),
		IssuedAt:       time.Date(2026, time.September, 30, 9, 29, 45, 0, model.Tbilisi()),
	}
}
