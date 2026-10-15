package service

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/dto"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

// CollectionStore holds the assignment, the transactions, and the bills of the acting
// workspace.
type CollectionStore interface {
	AssignedCustomer(ctx context.Context) (string, error)
	InsertTransactions(
		ctx context.Context,
		transactions []model.Transaction,
	) ([]model.Transaction, error)
	InsertBills(ctx context.Context, bills []model.Bill) error
	ChargesOn(
		ctx context.Context,
		customerNumber string,
		day time.Time,
	) ([]model.Transaction, error)
}

// Collector collects the transactions and the bills of the assigned customer of the
// acting workspace.
type Collector struct {
	account *account
	client  APIClientService
	store   CollectionStore
}

// NewCollector creates a new Collector.
func NewCollector(
	client APIClientService,
	settings pluginapi.PluginSettings,
	secrets pluginapi.PluginSecrets,
	store CollectionStore,
) *Collector {
	return &Collector{
		account: &account{client: client, settings: settings, secrets: secrets},
		client:  client,
		store:   store,
	}
}

// Collect reads the transactions and the bills of the assigned customer of the acting
// workspace in the month, and stores each one. It sends no bill. It answers
// errs.ErrNotAssigned with no assigned customer and errs.ErrNotConnected with no
// connection, and calls no portal for either.
func (c *Collector) Collect(ctx context.Context, month model.Month) (model.Collection, error) {
	customerNumber, err := c.store.AssignedCustomer(ctx)
	if err != nil {
		return model.Collection{}, fmt.Errorf("reading the assigned customer: %w", err)
	}

	session, err := c.account.session(ctx)
	if err != nil {
		return model.Collection{}, err
	}

	answered, err := c.read(ctx, session, customerNumber, month)
	if err != nil {
		return model.Collection{}, err
	}

	transactions, bills, err := mapMonth(answered, customerNumber)
	if err != nil {
		return model.Collection{}, err
	}

	stored, err := c.store.InsertTransactions(ctx, transactions)
	if err != nil {
		return model.Collection{}, fmt.Errorf("storing the transactions: %w", err)
	}

	if err = c.store.InsertBills(ctx, bills); err != nil {
		return model.Collection{}, fmt.Errorf("storing the bills: %w", err)
	}

	newest, err := c.newestReading(ctx, customerNumber, stored)
	if err != nil {
		return model.Collection{}, err
	}

	return model.Collection{Count: len(answered.payments), NewestReading: newest}, nil
}

// monthAnswer is what the portal answers for one month of one customer.
type monthAnswer struct {
	payments []dto.Payment
	receipts []dto.Receipt
}

// read checks that the account still holds the customer, then reads the month. Each
// call takes the session that the call before it answered.
func (c *Collector) read(
	ctx context.Context,
	session model.Session,
	customerNumber string,
	month model.Month,
) (monthAnswer, error) {
	accounts, session, err := callWith(ctx, c.account, session,
		func(session model.Session) ([]dto.Account, error) {
			return c.client.Accounts(ctx, session)
		})
	if err != nil {
		return monthAnswer{}, fmt.Errorf("reading the customers: %w", err)
	}

	if !slices.ContainsFunc(
		accounts,
		func(a dto.Account) bool { return a.Number == customerNumber },
	) {
		return monthAnswer{}, errs.ErrCustomerAbsent
	}

	payments, session, err := callWith(ctx, c.account, session,
		func(session model.Session) ([]dto.Payment, error) {
			return c.client.Payments(ctx, session, customerNumber, month)
		})
	if err != nil {
		return monthAnswer{}, fmt.Errorf("reading the transactions: %w", err)
	}

	receipts, _, err := callWith(ctx, c.account, session,
		func(session model.Session) ([]dto.Receipt, error) {
			return c.client.Receipts(ctx, session, customerNumber, month)
		})
	if err != nil {
		return monthAnswer{}, fmt.Errorf("reading the bills: %w", err)
	}

	return monthAnswer{payments: payments, receipts: receipts}, nil
}

// newestReading answers the reading of the latest date on which the collection stored a
// charge for the first time. It reads every stored charge of that date, because the
// other charge of a split reading can be one that an earlier collection stored.
func (c *Collector) newestReading(
	ctx context.Context,
	customerNumber string,
	stored []model.Transaction,
) (*model.Reading, error) {
	var latest *time.Time

	for i := range stored {
		if stored[i].IsCharge() && (latest == nil || stored[i].Date.After(*latest)) {
			latest = &stored[i].Date
		}
	}

	if latest == nil {
		return nil, nil //nolint:nilnil // no new charge is no failure
	}

	charges, err := c.store.ChargesOn(ctx, customerNumber, *latest)
	if err != nil {
		return nil, fmt.Errorf("reading the charges of the new reading: %w", err)
	}

	readings := model.ReadingsOf(charges)
	if len(readings) == 0 {
		return nil, nil //nolint:nilnil // a charge removed meanwhile is no failure
	}

	return &readings[0], nil
}

func mapMonth(
	answered monthAnswer,
	customerNumber string,
) ([]model.Transaction, []model.Bill, error) {
	transactions := make([]model.Transaction, 0, len(answered.payments))

	for i := range answered.payments {
		transaction, err := answered.payments[i].MapToModel(customerNumber)
		if err != nil {
			return nil, nil, fmt.Errorf("mapping a transaction of the customer: %w", err)
		}

		transactions = append(transactions, transaction)
	}

	bills := make([]model.Bill, 0, len(answered.receipts))

	for i := range answered.receipts {
		bill, err := answered.receipts[i].MapToModel(customerNumber)
		if err != nil {
			return nil, nil, fmt.Errorf("mapping a bill of the customer: %w", err)
		}

		bills = append(bills, bill)
	}

	return transactions, bills, nil
}
