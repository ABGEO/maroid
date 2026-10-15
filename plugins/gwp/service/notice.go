package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/gwp/dto"
	"github.com/abgeo/maroid/plugins/gwp/model"
)

const (
	// NewReadingTopic is the notification topic of a new reading of the water meters.
	NewReadingTopic = "new_reading"

	noticeTitle      = "GWP | წყლის ჩვენება"
	noticeDateLayout = "02/01/2006"
)

// errBalanceAbsent reports a list of the customers that does not hold the assigned one.
var errBalanceAbsent = errors.New("the list of the customers holds no assigned customer")

// DatedReadingStore reads the readings of one date of the acting workspace.
type DatedReadingStore interface {
	ReadingsOn(ctx context.Context, customerNumber string, date time.Time) ([]model.Reading, error)
}

// Notices sends the new readings of the assigned customer to the acting workspace.
type Notices struct {
	logger   *slog.Logger
	account  *account
	client   APIClientService
	notifier pluginapi.PluginNotifier
	store    DatedReadingStore
}

// NewNotices creates a new Notices.
func NewNotices(
	logger *slog.Logger,
	client APIClientService,
	settings pluginapi.PluginSettings,
	secrets pluginapi.PluginSecrets,
	notifier pluginapi.PluginNotifier,
	store DatedReadingStore,
) *Notices {
	return &Notices{
		logger: logger.With(
			slog.String("component", "service"),
			slog.String("service", "notices"),
		),
		account:  &account{client: client, settings: settings, secrets: secrets},
		client:   client,
		notifier: notifier,
		store:    store,
	}
}

// Send sends the readings of the customer on the date, with the balance of the customer.
// A balance that the water service fails to answer leaves the message without it,
// because the readings are stored and no later run sends them again.
func (n *Notices) Send(ctx context.Context, customerNumber string, date time.Time) error {
	readings, err := n.store.ReadingsOn(ctx, customerNumber, date)
	if err != nil {
		return fmt.Errorf("reading the new readings: %w", err)
	}

	err = n.notifier.Notify(ctx, NewReadingTopic, pluginapi.Message{
		Title: noticeTitle,
		Body:  noticeBody(date, readings, n.balance(ctx, customerNumber)),
	})
	if err != nil {
		return fmt.Errorf("sending the new readings: %w", err)
	}

	return nil
}

// balance answers the balance of the customer, or nil when the water service fails to
// answer it.
func (n *Notices) balance(ctx context.Context, customerNumber string) *float64 {
	balance, err := n.readBalance(ctx, customerNumber)
	if err != nil {
		n.logger.WarnContext(
			ctx,
			"sending the new readings without the balance",
			slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx)),
			slog.Any("error", err),
		)

		return nil
	}

	return &balance
}

func (n *Notices) readBalance(ctx context.Context, customerNumber string) (float64, error) {
	token, err := n.account.token(ctx)
	if err != nil {
		return 0, err
	}

	var customers []dto.Customer

	err = n.account.call(ctx, token, func(token string) error {
		var listErr error

		customers, listErr = n.client.GetCustomers(ctx, token)
		if listErr != nil {
			return fmt.Errorf("requesting the customers: %w", listErr)
		}

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("reading the balance: %w", err)
	}

	for _, customer := range customers {
		if customer.CustomerNumber == customerNumber {
			return customer.Balance, nil
		}
	}

	return 0, errBalanceAbsent
}

func noticeBody(date time.Time, readings []model.Reading, balance *float64) string {
	var body strings.Builder

	body.WriteString("წყლის მრიცხველის ახალი ჩვენება.\n\n")
	fmt.Fprintf(&body, "<b>თარიღი</b>: %s\n", date.Format(noticeDateLayout))

	for _, reading := range readings {
		fmt.Fprintf(&body, "<b>%s</b>: %.1f მ³", reading.SerialNumber, reading.Value)

		if reading.Usage != nil {
			fmt.Fprintf(&body, ", მოხმარება %.1f მ³", *reading.Usage)
		}

		body.WriteString("\n")
	}

	if balance == nil {
		body.WriteString("<b>ბალანსი</b>: მიუწვდომელია")
	} else {
		fmt.Fprintf(&body, "<b>ბალანსი</b>: %.2f ₾", *balance)
	}

	return body.String()
}
