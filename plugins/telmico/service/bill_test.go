package service_test

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/model"
	"github.com/abgeo/maroid/plugins/telmico/service"
)

// fakeSender records each notification.
type fakeSender struct {
	mu       sync.Mutex
	topics   []string
	messages []pluginapi.Message
}

func (f *fakeSender) Notify(
	_ context.Context,
	topicID string,
	msg pluginapi.Message,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.topics = append(f.topics, topicID)
	f.messages = append(f.messages, msg)

	return nil
}

func (f *fakeSender) sent() ([]string, []pluginapi.Message) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.topics...), append([]pluginapi.Message(nil), f.messages...)
}

func billsOf(
	portal *fakeClient,
	secrets *memorySecrets,
	store *memoryStore,
) (*service.Bills, *fakeSender, *bytes.Buffer) {
	sender := new(fakeSender)
	logs := new(bytes.Buffer)

	bills := service.NewBills(
		slog.New(slog.NewTextHandler(logs, nil)),
		portal,
		storedCredentials{},
		secrets,
		store,
		sender,
	)

	return bills, sender, logs
}

func splitReading() model.Reading {
	return model.Reading{
		CustomerNumber: customerC1,
		Date:           time.Date(2026, time.April, 28, 0, 0, 0, 0, model.Tbilisi()),
		Usage:          13.2 + 153.3,
		MeterValue:     1317,
		Charge:         2.52 + 36.87,
	}
}

func storeWithAprilBill() *memoryStore {
	store := storeAssigning(customerC1)
	store.bills["8370764-2026-4"] = model.Bill{
		CustomerNumber: customerC1,
		BillNumber:     "8370764-2026-4",
		ServiceBillID:  uuid.MustParse("77fc2c0c-4509-43b2-adf3-07185abffe78"),
		Month:          model.Month{Year: 2026, Month: time.April}.First(),
	}

	return store
}

// ELECACC-SC-021: The bill of a split reading shows the date, the highest meter value,
// the summed usage, and the summed charge, with the document of its month attached.
func TestTheBillShowsTheReadingWithItsDocument(t *testing.T) {
	t.Parallel()

	portal := portalAccepting(cookieK)
	bills, sender, _ := billsOf(portal, secretsHolding(new(session())), storeWithAprilBill())

	require.NoError(t, bills.Send(actingInW(t), splitReading()))

	topics, messages := sender.sent()
	require.Equal(t, []string{service.MonthlyBillTopic}, topics)
	require.Len(t, messages, 1)

	message := messages[0]
	assert.Equal(t, "თელმიკო | ქვითარი", message.Title)
	assert.Contains(t, message.Body, "2026-04-28")
	assert.Contains(t, message.Body, "1317 კვტ/სთ")
	assert.Contains(t, message.Body, "166.5 კვტ/სთ")
	assert.Contains(t, message.Body, "39.39 ₾")

	require.Len(t, message.Attachments, 1)
	assert.Equal(t, "8370764-2026-4.pdf", message.Attachments[0].Filename)
	assert.Equal(t, "application/pdf", message.Attachments[0].MIMEType)
	assert.Equal(t, []byte("%PDF-1.3"), message.Attachments[0].Content)
}

// ELECACC-SC-022: A document that fails to download, or a month with no bill, leaves
// the message without it and logs a warning.
func TestTheBillGoesWithoutADocumentThatFails(t *testing.T) {
	t.Parallel()

	failing := portalAccepting(cookieK)
	failing.documentFails = true

	cases := map[string]struct {
		portal *fakeClient
		store  *memoryStore
	}{
		"download fails": {portal: failing, store: storeWithAprilBill()},
		"no bill":        {portal: portalAccepting(cookieK), store: storeAssigning(customerC1)},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			bills, sender, logs := billsOf(
				setup.portal,
				secretsHolding(new(session())),
				setup.store,
			)

			require.NoError(t, bills.Send(actingInW(t), splitReading()))

			_, messages := sender.sent()
			require.Len(t, messages, 1)
			assert.Empty(t, messages[0].Attachments)
			assert.Contains(t, logs.String(), "level=WARN")
		})
	}
}

// ELECACC-SC-008: The download of the document signs in again when the session ended.
func TestTheBillDownloadSignsInAgain(t *testing.T) {
	t.Parallel()

	portal := portalAccepting(cookieK)
	secrets := secretsHolding(&model.Session{Cookie: cookieK0, CSRFToken: csrfT})
	bills, sender, _ := billsOf(portal, secrets, storeWithAprilBill())

	require.NoError(t, bills.Send(actingInW(t), splitReading()))

	_, messages := sender.sent()
	require.Len(t, messages, 1)
	assert.Len(t, messages[0].Attachments, 1)

	_, _, signIns := portal.recorded()
	assert.Equal(t, 1, signIns)
}
