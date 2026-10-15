package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
)

const (
	// MonthlyBillTopic is the notification topic of the bill of the last month.
	MonthlyBillTopic = "monthly_bill"

	billTitle      = "თბილისი ენერჯი | ქვითარი"
	billDateLayout = "2006.01.02"
)

// Bills sends the bill of a reading to the acting workspace.
type Bills struct {
	logger   *slog.Logger
	account  *account
	client   APIClientService
	notifier pluginapi.PluginNotifier
}

// NewBills creates a new Bills.
func NewBills(
	logger *slog.Logger,
	client APIClientService,
	settings pluginapi.PluginSettings,
	secrets pluginapi.PluginSecrets,
	notifier pluginapi.PluginNotifier,
) *Bills {
	return &Bills{
		logger:   logger.With(slog.String("component", "service"), slog.String("service", "bills")),
		account:  &account{client: client, settings: settings, secrets: secrets},
		client:   client,
		notifier: notifier,
	}
}

// Send sends the bill of the reading, with the bill and the meter photo attached. A
// file that fails to download leaves the bill without it, because the reading is
// stored and no later run sends its bill again.
func (b *Bills) Send(ctx context.Context, reading model.Transaction) error {
	err := b.notifier.Notify(ctx, MonthlyBillTopic, pluginapi.Message{
		Title:       billTitle,
		Body:        billBody(reading),
		Attachments: b.attachments(ctx, reading),
	})
	if err != nil {
		return fmt.Errorf("sending the bill: %w", err)
	}

	return nil
}

func (b *Bills) attachments(ctx context.Context, reading model.Transaction) []pluginapi.Attachment {
	date := reading.Date.Format(billDateLayout)
	files := []struct {
		path     *string
		filename string
		mimeType string
	}{
		{reading.BillPath, "bill_" + date + ".pdf", "application/pdf"},
		{reading.MeterPhotoPath, "meter_photo_" + date + ".jpg", "image/jpeg"},
	}

	token, err := b.account.token(ctx)
	if err != nil {
		b.warn(ctx, "sending the bill with no file, because the token is unreadable", err)

		return nil
	}

	var attachments []pluginapi.Attachment

	for _, file := range files {
		if file.path == nil {
			continue
		}

		var content []byte

		next, err := b.account.call(ctx, token, func(token string) error {
			var downloadErr error

			content, downloadErr = b.client.DownloadFile(ctx, token, *file.path)
			if downloadErr != nil {
				return fmt.Errorf("downloading %s: %w", file.filename, downloadErr)
			}

			return nil
		})
		if err != nil {
			b.warn(ctx, "sending the bill without a file", err, slog.String("file", file.filename))

			continue
		}

		token = next

		attachments = append(attachments, pluginapi.Attachment{
			Filename: file.filename,
			Content:  content,
			MIMEType: file.mimeType,
		})
	}

	return attachments
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

func billBody(reading model.Transaction) string {
	return fmt.Sprintf(`ბუნებრივი აირის მოხმარების ყოველთვიური ქვითარი.

<b>თარიღი</b>: %s
<b>მრიცხველის ჩვენება</b>: %.0f მ³
<b>მოხმარება</b>: %.0f მ³
<b>სულ გადასახადი</b>: %.2f ₾`,
		reading.Date.Format(billDateLayout),
		reading.MeterValue,
		reading.Consumption,
		reading.Amount,
	)
}
