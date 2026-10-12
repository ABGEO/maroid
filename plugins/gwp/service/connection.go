package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/gwp/dto"
	"github.com/abgeo/maroid/plugins/gwp/errs"
	"github.com/abgeo/maroid/plugins/gwp/model"
)

// AssignmentStore holds the assigned customer of the acting workspace.
type AssignmentStore interface {
	AssignedCustomer(ctx context.Context) (string, error)
	Assign(ctx context.Context, customerNumber string) error
	Unassign(ctx context.Context) error
}

// ReadingCollector collects the readings of the assigned customer of the acting workspace.
type ReadingCollector interface {
	Collect(ctx context.Context) (int, error)
}

// Connections connects the water account of the acting workspace, and assigns one of
// its customers to the workspace, over the API client, the settings, the plugin
// secrets, and the store of the assignment.
type Connections struct {
	logger    *slog.Logger
	account   *account
	client    APIClientService
	store     AssignmentStore
	collector ReadingCollector
}

// NewConnections creates a new Connections instance.
func NewConnections(
	logger *slog.Logger,
	client APIClientService,
	settings *pluginapi.PluginSettings,
	secrets *pluginapi.PluginSecrets,
	store AssignmentStore,
	collector ReadingCollector,
) *Connections {
	return &Connections{
		logger:    logger,
		account:   &account{client: client, settings: settings, secrets: secrets},
		client:    client,
		store:     store,
		collector: collector,
	}
}

// Status reads the connection of the acting workspace. A connected workspace reads
// the customers of the account, which also proves that the token still works. A
// token that the water service rejects is removed, and the workspace reads as not
// connected.
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
		return nil, fmt.Errorf("%w: the water service rejected a new token", errs.ErrRequestFailed)
	}

	if err != nil {
		return nil, fmt.Errorf("listing the customers with a new token: %w", err)
	}

	return c.connection(ctx, customers, true)
}

// Disconnect removes the token and the assignment of the acting workspace. The
// settings and the readings stay.
func (c *Connections) Disconnect(ctx context.Context) error {
	if err := c.account.forgetToken(ctx); err != nil {
		return err
	}

	if err := c.store.Unassign(ctx); err != nil {
		return fmt.Errorf("removing the assignment: %w", err)
	}

	return nil
}

// Assign assigns a customer of the account to the acting workspace, then collects
// its readings. A failed collection keeps the assignment, because the scheduled
// collection retries it.
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

	if _, err = c.collector.Collect(ctx); err != nil {
		c.logger.WarnContext(
			ctx,
			"the collection after an assignment failed, the scheduled collection retries it",
			slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx)),
			slog.Any("error", err),
		)
	}

	return connectionOf(customers, customerNumber, credentialsStored), nil
}

// customers lists the customers of the account. A token that the water service
// rejects is removed, and the answer is errs.ErrNotConnected.
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

	for _, customer := range customers {
		connection.Customers = append(connection.Customers, model.Customer{
			Number:         customer.CustomerNumber,
			CommunalNumber: strings.TrimSpace(customer.CommunalNumber),
			Address:        addressOf(customer),
			Assigned:       customer.CustomerNumber == assigned,
		})
	}

	return connection
}

func holdsCustomer(customers []dto.Customer, customerNumber string) bool {
	for _, customer := range customers {
		if customer.CustomerNumber == customerNumber {
			return true
		}
	}

	return false
}

func addressOf(customer dto.Customer) string {
	fields := []string{customer.City, customer.Street, customer.Building}
	parts := make([]string, 0, len(fields))

	for _, part := range fields {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}

	return strings.Join(parts, ", ")
}
