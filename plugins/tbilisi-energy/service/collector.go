package service

import (
	"context"
	"fmt"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/dto"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/errs"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
)

// TransactionStore holds the assignment and the transactions of the acting workspace.
type TransactionStore interface {
	AssignedCustomer(ctx context.Context) (string, error)
	InsertTransactions(
		ctx context.Context,
		transactions []model.Transaction,
	) ([]model.Transaction, error)
}

// Collector collects the transactions of the assigned customer of the acting workspace.
type Collector struct {
	account *account
	client  APIClientService
	store   TransactionStore
}

// NewCollector creates a new Collector.
func NewCollector(
	client APIClientService,
	settings pluginapi.PluginSettings,
	secrets pluginapi.PluginSecrets,
	store TransactionStore,
) *Collector {
	return &Collector{
		account: &account{client: client, settings: settings, secrets: secrets},
		client:  client,
		store:   store,
	}
}

// Collect reads the transactions of the assigned customer of the acting workspace in
// the period, and stores each one. It sends no bill. It answers errs.ErrNotAssigned
// with no assigned customer and errs.ErrNotConnected with no connection.
func (c *Collector) Collect(ctx context.Context, period model.Period) (model.Collection, error) {
	customerNumber, err := c.store.AssignedCustomer(ctx)
	if err != nil {
		return model.Collection{}, fmt.Errorf("reading the assigned customer: %w", err)
	}

	token, err := c.account.token(ctx)
	if err != nil {
		return model.Collection{}, err
	}

	token, err = c.ensureCustomer(ctx, token, customerNumber)
	if err != nil {
		return model.Collection{}, err
	}

	var answered []dto.Transaction

	_, err = c.account.call(ctx, token, func(token string) error {
		var getErr error

		answered, getErr = c.client.GetTransactions(
			ctx, token, dto.NewTransactionsRequest(customerNumber, period),
		)
		if getErr != nil {
			return fmt.Errorf("requesting the transactions of the period: %w", getErr)
		}

		return nil
	})
	if err != nil {
		return model.Collection{}, fmt.Errorf("reading the transactions: %w", err)
	}

	transactions, err := transactionsOf(answered, customerNumber)
	if err != nil {
		return model.Collection{}, err
	}

	stored, err := c.store.InsertTransactions(ctx, transactions)
	if err != nil {
		return model.Collection{}, fmt.Errorf("storing the transactions: %w", err)
	}

	return model.Collection{Count: len(answered), NewestReading: newestReading(stored)}, nil
}

// ensureCustomer answers errs.ErrCustomerAbsent when the account no longer holds the
// customer. It answers the token that the next call takes.
func (c *Collector) ensureCustomer(
	ctx context.Context,
	token string,
	customerNumber string,
) (string, error) {
	var customers []dto.Customer

	token, err := c.account.call(ctx, token, func(token string) error {
		var getErr error

		customers, getErr = c.client.GetCustomers(ctx, token)
		if getErr != nil {
			return fmt.Errorf("requesting the customers: %w", getErr)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("reading the customers: %w", err)
	}

	if !holdsCustomer(customers, customerNumber) {
		return "", errs.ErrCustomerAbsent
	}

	return token, nil
}

func holdsCustomer(customers []dto.Customer, customerNumber string) bool {
	for i := range customers {
		if customers[i].Number() == customerNumber {
			return true
		}
	}

	return false
}

func transactionsOf(
	answered []dto.Transaction,
	customerNumber string,
) ([]model.Transaction, error) {
	transactions := make([]model.Transaction, 0, len(answered))

	for i := range answered {
		transaction, err := answered[i].MapToModel(customerNumber)
		if err != nil {
			return nil, fmt.Errorf("mapping a transaction of the customer: %w", err)
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func newestReading(transactions []model.Transaction) *model.Transaction {
	var newest *model.Transaction

	for i := range transactions {
		if !transactions[i].IsReading() {
			continue
		}

		if newest == nil || transactions[i].Date.After(newest.Date) {
			newest = &transactions[i]
		}
	}

	return newest
}
