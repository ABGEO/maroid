package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/dto"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/errs"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
)

// AssignmentStore holds the assigned customer of the acting workspace.
type AssignmentStore interface {
	AssignedCustomer(ctx context.Context) (string, error)
	Assign(ctx context.Context, customerNumber string) error
	Unassign(ctx context.Context) error
}

// TransactionCollector collects the transactions of the acting workspace in a period.
type TransactionCollector interface {
	Collect(ctx context.Context, period model.Period) (model.Collection, error)
}

// BillSender sends the bill of a reading to the acting workspace.
type BillSender interface {
	Send(ctx context.Context, reading model.Transaction) error
}

// Connections connects the gas account of the acting workspace, and assigns one of its
// customers to the workspace.
type Connections struct {
	logger    *slog.Logger
	account   *account
	client    APIClientService
	store     AssignmentStore
	collector TransactionCollector
	bills     BillSender
	now       func() time.Time
}

// NewConnections creates a new Connections instance. now gives the time of an
// assignment, which picks the month that the assignment collects.
func NewConnections(
	logger *slog.Logger,
	client APIClientService,
	settings *pluginapi.PluginSettings,
	secrets *pluginapi.PluginSecrets,
	store AssignmentStore,
	collector TransactionCollector,
	bills BillSender,
	now func() time.Time,
) *Connections {
	return &Connections{
		logger:    logger,
		account:   &account{client: client, settings: settings, secrets: secrets},
		client:    client,
		store:     store,
		collector: collector,
		bills:     bills,
		now:       now,
	}
}

// Status reads the connection of the acting workspace. A connected workspace reads the
// customers of the account, which also proves that the token still works. A token that
// the gas service rejects is removed, and the workspace reads as not connected.
func (c *Connections) Status(ctx context.Context) (*model.Connection, error) {
	credentialsStored, err := c.credentialsStored(ctx)
	if err != nil {
		return nil, err
	}

	notConnected := &model.Connection{
		State:             model.ConnectionStateNotConnected,
		CredentialsStored: credentialsStored,
	}

	token, err := c.account.token(ctx)
	if errors.Is(err, errs.ErrNotConnected) {
		return notConnected, nil
	}

	if err != nil {
		return nil, err
	}

	customers, err := c.customers(ctx, token)
	if errors.Is(err, errs.ErrNotConnected) {
		return notConnected, nil
	}

	if err != nil {
		return nil, err
	}

	return c.connection(ctx, customers, credentialsStored)
}

// Connect signs in with the stored credentials and stores the token. The new token
// replaces the old one, and the assigned customer stays.
func (c *Connections) Connect(ctx context.Context) (*model.Connection, error) {
	token, err := c.account.signIn(ctx)
	if err != nil {
		return nil, err
	}

	customers, err := c.client.GetCustomers(ctx, token)
	if errors.Is(err, errs.ErrNotConnected) {
		return nil, fmt.Errorf("%w: the gas service rejected a new token", errs.ErrRequestFailed)
	}

	if err != nil {
		return nil, fmt.Errorf("listing the customers with a new token: %w", err)
	}

	return c.connection(ctx, customers, true)
}

// Disconnect removes the token and the assignment of the acting workspace. The
// settings and the transactions stay.
func (c *Connections) Disconnect(ctx context.Context) error {
	if err := c.account.forgetToken(ctx); err != nil {
		return err
	}

	if err := c.store.Unassign(ctx); err != nil {
		return fmt.Errorf("removing the assignment: %w", err)
	}

	return nil
}

// Assign assigns a customer of the account to the acting workspace, then collects the
// previous month and sends the bill of its new reading. A failed collection or bill
// keeps the assignment, because the scheduled collection retries it.
func (c *Connections) Assign(
	ctx context.Context,
	customerNumber string,
) (*model.Connection, error) {
	token, err := c.account.token(ctx)
	if err != nil {
		return nil, err
	}

	customers, err := c.customers(ctx, token)
	if err != nil {
		return nil, err
	}

	credentialsStored, err := c.credentialsStored(ctx)
	if err != nil {
		return nil, err
	}

	if !holdsCustomer(customers, customerNumber) {
		return nil, errs.ErrCustomerUnknown
	}

	if err = c.store.Assign(ctx, customerNumber); err != nil {
		return nil, fmt.Errorf("storing the assignment: %w", err)
	}

	c.collectPreviousMonth(ctx)

	return connectionOf(customers, customerNumber, credentialsStored), nil
}

func (c *Connections) collectPreviousMonth(ctx context.Context) {
	collection, err := c.collector.Collect(ctx, model.PreviousMonth(c.now()))
	if err == nil && collection.NewestReading != nil {
		err = c.bills.Send(ctx, *collection.NewestReading)
	}

	if err != nil {
		c.logger.WarnContext(
			ctx,
			"the collection after an assignment failed, the scheduled collection retries it",
			slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx)),
			slog.Any("error", err),
		)
	}
}

// customers lists the customers of the account. A token that the gas service rejects
// is removed, and the answer is errs.ErrNotConnected.
func (c *Connections) customers(ctx context.Context, token string) ([]dto.Customer, error) {
	customers, err := c.client.GetCustomers(ctx, token)
	if errors.Is(err, errs.ErrNotConnected) {
		if forgetErr := c.account.forgetToken(ctx); forgetErr != nil {
			return nil, forgetErr
		}

		return nil, errs.ErrNotConnected
	}

	if err != nil {
		return nil, fmt.Errorf("listing the customers: %w", err)
	}

	return customers, nil
}

func (c *Connections) connection(
	ctx context.Context,
	customers []dto.Customer,
	credentialsStored bool,
) (*model.Connection, error) {
	assigned, err := c.store.AssignedCustomer(ctx)
	if err != nil && !errors.Is(err, errs.ErrNotAssigned) {
		return nil, fmt.Errorf("reading the assignment: %w", err)
	}

	return connectionOf(customers, assigned, credentialsStored), nil
}

func (c *Connections) credentialsStored(ctx context.Context) (bool, error) {
	_, err := c.account.credentials(ctx)

	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, errs.ErrCredentialsAbsent):
		return false, nil
	default:
		return false, err
	}
}

func connectionOf(
	customers []dto.Customer,
	assigned string,
	credentialsStored bool,
) *model.Connection {
	connection := &model.Connection{
		State:             model.ConnectionStateConnected,
		CredentialsStored: credentialsStored,
		Customers:         make([]model.Customer, 0, len(customers)),
	}

	for i := range customers {
		number := customers[i].Number()

		connection.Customers = append(connection.Customers, model.Customer{
			Number:   number,
			Name:     strings.TrimSpace(customers[i].Name),
			Address:  strings.TrimSpace(customers[i].Address),
			Assigned: number == assigned,
		})
	}

	return connection
}
