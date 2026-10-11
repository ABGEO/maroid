package telegram_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/notification"
	"github.com/abgeo/maroid/apps/hub/internal/notification/telegram"
	"github.com/abgeo/maroid/apps/hub/internal/notification/telegramtest"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	token        = "123456:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	tokenField   = "bot_token"
	sendMessage  = "sendMessage"
	chatField    = "chat_id"
	threadField  = "thread_id"
	pngSignature = "\x89PNG\r\n\x1a\n0000"
)

func methodsOf(calls []telegramtest.Call) []string {
	methods := make([]string, 0, len(calls))
	for _, call := range calls {
		methods = append(methods, call.Method)
	}

	return methods
}

type shapeCase struct {
	attachments []pluginapi.Attachment
	methods     []string
}

// shapeCases holds the rows of the table of NOTIF-SC-023.
func shapeCases() map[string]shapeCase {
	image := pluginapi.Attachment{
		Filename: "chart.png",
		Content:  []byte(pngSignature),
		MIMEType: "image/png",
	}
	pdf := pluginapi.Attachment{
		Filename: "bill.pdf",
		Content:  []byte("%PDF-1.4"),
		MIMEType: "application/pdf",
	}

	return map[string]shapeCase{
		"no attachment": {methods: []string{sendMessage}},
		"one image": {
			attachments: []pluginapi.Attachment{image},
			methods:     []string{"sendPhoto"},
		},
		"one document": {
			attachments: []pluginapi.Attachment{pdf},
			methods:     []string{"sendDocument"},
		},
		"mixed": {
			attachments: []pluginapi.Attachment{image, image, pdf},
			methods:     []string{sendMessage, "sendMediaGroup", "sendMediaGroup"},
		},
	}
}

// NOTIF-SC-023: Telegram receives the title, the body, and every attachment, in
// the thread that the destination names.
func TestSendDeliversEachShapeOfMessage(t *testing.T) {
	t.Parallel()

	cases := shapeCases()

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stub := telegramtest.NewServer(t)
			kind := telegram.New(telego.WithAPIServer(stub.URL()))

			err := kind.Send(
				t.Context(),
				map[string]any{tokenField: token},
				map[string]any{chatField: "-100", threadField: "2749"},
				pluginapi.Message{
					Title:       "Plant care due",
					Body:        "E001 Office",
					Attachments: tc.attachments,
				},
			)
			require.NoError(t, err)

			calls := stub.Calls()
			require.Equal(t, tc.methods, methodsOf(calls))

			for _, call := range calls {
				require.Equal(t, "-100", call.Params[chatField])
				require.Equal(t, "2749", call.Params["message_thread_id"])
			}

			if calls[0].Method == sendMessage {
				require.Equal(t, "<b>Plant care due</b>\n\nE001 Office", calls[0].Params["text"])
			}
		})
	}
}

// NOTIF-SC-023: A destination with no thread sends to the chat itself.
func TestSendWithNoThreadNamesNoThread(t *testing.T) {
	t.Parallel()

	stub := telegramtest.NewServer(t)
	kind := telegram.New(telego.WithAPIServer(stub.URL()))

	require.NoError(t, kind.Send(
		t.Context(),
		map[string]any{tokenField: token},
		map[string]any{chatField: "-100"},
		pluginapi.Message{Title: "Plant care due"},
	))

	calls := stub.Calls()
	require.Len(t, calls, 1)
	require.NotContains(t, calls[0].Params, "message_thread_id")
}

// NOTIF-SC-010: The check of a token tells an accepted token, a refused token, and a
// Telegram that does not answer.
func TestVerifyClassifiesTheAnswerOfTelegram(t *testing.T) {
	t.Parallel()

	t.Run("accepted", func(t *testing.T) {
		t.Parallel()

		stub := telegramtest.NewServer(t)
		kind := telegram.New(telego.WithAPIServer(stub.URL()))

		require.NoError(t, kind.Verify(t.Context(), map[string]any{tokenField: token}))
		require.Equal(t, []string{"getMe"}, methodsOf(stub.Calls()))
	})

	t.Run("refused", func(t *testing.T) {
		t.Parallel()

		stub := telegramtest.NewServer(t)
		stub.Refuse("", "Unauthorized")
		kind := telegram.New(telego.WithAPIServer(stub.URL()))

		err := kind.Verify(t.Context(), map[string]any{tokenField: token})

		require.ErrorIs(t, err, errs.ErrDeliveryRefused)

		refusal, ok := errors.AsType[*notification.RefusedError](err)
		require.True(t, ok)
		require.Equal(t, "Unauthorized", refusal.Reason)
	})

	t.Run("no answer", func(t *testing.T) {
		t.Parallel()

		stub := telegramtest.NewServer(t)
		stub.Silence()
		kind := telegram.New(telego.WithAPIServer(stub.URL()))

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		err := kind.Verify(ctx, map[string]any{tokenField: token})

		require.ErrorIs(t, err, errs.ErrMessagingUnreachable)
	})
}
