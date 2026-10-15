package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/dto"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

// AssignmentStore holds the assigned customer of the acting workspace.
type AssignmentStore interface {
	AssignedCustomer(ctx context.Context) (string, error)
	Assign(ctx context.Context, customerNumber string) error
	Unassign(ctx context.Context) error
}

// Connections connects the electricity account of the acting workspace, and assigns
// one of its customers to the workspace. It collects nothing: an administrator collects
// the history from the command line, and the schedule collects each month.
type Connections struct {
	account *account
	client  APIClientService
	store   AssignmentStore
}

// NewConnections creates a new Connections instance.
func NewConnections(
	client APIClientService,
	settings pluginapi.PluginSettings,
	secrets pluginapi.PluginSecrets,
	store AssignmentStore,
) *Connections {
	return &Connections{
		account: &account{client: client, settings: settings, secrets: secrets},
		client:  client,
		store:   store,
	}
}

// Status reads the connection of the acting workspace. A connected workspace reads the
// customers of the account, which also proves that the session still works. A session
// that no sign in can renew reads as not connected.
func (c *Connections) Status(ctx context.Context) (*model.Connection, error) {
	credentialsStored, err := c.credentialsStored(ctx)
	if err != nil {
		return nil, err
	}

	notConnected := &model.Connection{
		State:             model.ConnectionStateNotConnected,
		CredentialsStored: credentialsStored,
	}

	session, err := c.account.session(ctx)
	if errors.Is(err, errs.ErrNotConnected) {
		return notConnected, nil
	}

	if err != nil {
		return nil, err
	}

	customers, err := c.customers(ctx, session)
	if isRejection(err) {
		return notConnected, nil
	}

	if err != nil {
		return nil, err
	}

	return c.connection(ctx, customers, credentialsStored)
}

// Connect signs in with the stored credentials and stores the session. The new session
// replaces the old one, and the assigned customer stays. A rejection answers as the
// credentials it names, because the editor asked for a sign in.
func (c *Connections) Connect(ctx context.Context) (*model.Connection, error) {
	session, err := c.account.signIn(ctx)
	if err != nil {
		return nil, err
	}

	customers, err := c.customers(ctx, session)
	if err != nil {
		return nil, err
	}

	return c.connection(ctx, customers, true)
}

// Disconnect removes the session and the assignment of the acting workspace. The
// settings, the transactions, and the bills stay.
func (c *Connections) Disconnect(ctx context.Context) error {
	if err := c.account.forgetSession(ctx); err != nil {
		return err
	}

	if err := c.store.Unassign(ctx); err != nil {
		return fmt.Errorf("removing the assignment: %w", err)
	}

	return nil
}

// Assign assigns a customer of the account to the acting workspace, and collects nothing.
func (c *Connections) Assign(
	ctx context.Context,
	customerNumber string,
) (*model.Connection, error) {
	session, err := c.account.session(ctx)
	if err != nil {
		return nil, err
	}

	customers, err := c.customers(ctx, session)
	if isRejection(err) {
		return nil, fmt.Errorf("%w: %w", errs.ErrNotConnected, err)
	}

	if err != nil {
		return nil, err
	}

	credentialsStored, err := c.credentialsStored(ctx)
	if err != nil {
		return nil, err
	}

	if !slices.ContainsFunc(
		customers,
		func(held model.Customer) bool { return held.Number == customerNumber },
	) {
		return nil, errs.ErrCustomerUnknown
	}

	if err = c.store.Assign(ctx, customerNumber); err != nil {
		return nil, fmt.Errorf("storing the assignment: %w", err)
	}

	return connectionOf(customers, customerNumber, credentialsStored), nil
}

// customers lists the customers of the account with the address of each. Credentials
// that the portal rejects at a new sign in remove the session.
func (c *Connections) customers(
	ctx context.Context,
	session model.Session,
) ([]model.Customer, error) {
	accounts, session, err := callWith(ctx, c.account, session,
		func(session model.Session) ([]dto.Account, error) {
			return c.client.Accounts(ctx, session)
		})
	if err != nil {
		return nil, fmt.Errorf("listing the customers: %w", err)
	}

	customers := make([]model.Customer, 0, len(accounts))

	for _, held := range accounts {
		var detail dto.AccountDetail

		detail, session, err = callWith(ctx, c.account, session,
			func(session model.Session) (dto.AccountDetail, error) {
				return c.client.AccountDetail(ctx, session, held.Number)
			})
		if err != nil {
			return nil, fmt.Errorf("reading the address of a customer: %w", err)
		}

		customers = append(customers, model.Customer{
			Number:  held.Number,
			Name:    strings.TrimSpace(held.Name),
			Address: strings.TrimSpace(detail.Address),
		})
	}

	return customers, nil
}

func (c *Connections) connection(
	ctx context.Context,
	customers []model.Customer,
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

// isRejection reports a call whose session no sign in renews: the portal rejected the
// stored credentials, or the workspace stores none. The rejection removed the session.
func isRejection(err error) bool {
	return errors.Is(err, errs.ErrCredentialsRejected) || errors.Is(err, errs.ErrCredentialsAbsent)
}

func connectionOf(
	customers []model.Customer,
	assigned string,
	credentialsStored bool,
) *model.Connection {
	connection := &model.Connection{
		State:             model.ConnectionStateConnected,
		CredentialsStored: credentialsStored,
		Customers:         make([]model.Customer, 0, len(customers)),
	}

	for _, customer := range customers {
		customer.Assigned = customer.Number == assigned
		connection.Customers = append(connection.Customers, customer)
	}

	return connection
}
