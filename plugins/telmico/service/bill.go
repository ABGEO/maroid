package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

const (
	// MonthlyBillTopic is the notification topic of the bill of the last month.
	MonthlyBillTopic = "monthly_bill"

	billTitle = "თელმიკო | ქვითარი"
)

// BillLookup finds the stored bill of a month of a customer.
type BillLookup interface {
	BillByMonth(ctx context.Context, customerNumber string, month model.Month) (*model.Bill, error)
}

// Bills sends the bill of a reading to the acting workspace.
type Bills struct {
	logger   *slog.Logger
	account  *account
	client   APIClientService
	store    BillLookup
	notifier *pluginapi.PluginNotifier
}

// NewBills creates a new Bills.
func NewBills(
	logger *slog.Logger,
	client APIClientService,
	settings *pluginapi.PluginSettings,
	secrets *pluginapi.PluginSecrets,
	store BillLookup,
	notifier *pluginapi.PluginNotifier,
) *Bills {
	return &Bills{
		logger:   logger.With(slog.String("component", "service"), slog.String("service", "bills")),
		account:  &account{client: client, settings: settings, secrets: secrets},
		client:   client,
		store:    store,
		notifier: notifier,
	}
}

// Send sends the bill of the reading, with the document of its month attached. A
// document that fails to download leaves the bill without it, because the reading is
// stored and no later run sends its bill again.
func (b *Bills) Send(ctx context.Context, reading model.Reading) error {
	message := pluginapi.Message{Title: billTitle, Body: billBody(reading)}

	if document, found := b.document(ctx, reading); found {
		message.Attachments = []pluginapi.Attachment{document}
	}

	if err := b.notifier.Notify(ctx, MonthlyBillTopic, message); err != nil {
		return fmt.Errorf("sending the bill: %w", err)
	}

	return nil
}

func (b *Bills) document(ctx context.Context, reading model.Reading) (pluginapi.Attachment, bool) {
	month := model.MonthOf(reading.Date)

	bill, err := b.store.BillByMonth(ctx, reading.CustomerNumber, month)
	if err != nil {
		b.warn(ctx, "sending the bill with no document, because the bill is unreadable", err)

		return pluginapi.Attachment{}, false
	}

	if bill == nil {
		b.warn(ctx, "sending the bill with no document, because the month holds no bill", nil,
			slog.String("month", month.String()))

		return pluginapi.Attachment{}, false
	}

	session, err := b.account.session(ctx)
	if err != nil {
		b.warn(ctx, "sending the bill with no document, because the session is unreadable", err)

		return pluginapi.Attachment{}, false
	}

	content, _, err := callWith(
		ctx,
		b.account,
		session,
		func(session model.Session) ([]byte, error) {
			return b.client.ReceiptDocument(ctx, session, bill.ServiceBillID)
		},
	)
	if err != nil {
		b.warn(ctx, "sending the bill with no document", err, slog.String("bill", bill.BillNumber))

		return pluginapi.Attachment{}, false
	}

	return pluginapi.Attachment{
		Filename: bill.BillNumber + ".pdf",
		Content:  content,
		MIMEType: mediaPDF,
	}, true
}

func (b *Bills) warn(ctx context.Context, message string, err error, attributes ...any) {
	b.logger.WarnContext(
		ctx,
		message,
		append([]any{
			slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx)),
			slog.Any("error", err),
		}, attributes...)...,
	)
}

func billBody(reading model.Reading) string {
	return fmt.Sprintf(`ელექტრო ენერგიის მოხმარების ყოველთვიური ქვითარი.

<b>თარიღი</b>: %s
<b>მრიცხველის ჩვენება</b>: %s კვტ/სთ
<b>მოხმარება</b>: %s კვტ/სთ
<b>სულ გადასახადი</b>: %.2f ₾`,
		reading.Date.Format(time.DateOnly),
		model.KilowattHours(reading.MeterValue),
		model.KilowattHours(reading.Usage),
		reading.Charge,
	)
}
