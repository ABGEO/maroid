package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	pluginOfMessages = "dev.maroid.p"
	pluginOfQ        = "dev.maroid.q"
	chatOfNina       = int64(7001)
	chatOfBeka       = int64(7002)
	lifetimeAndADay  = 31 * 24 * time.Hour
)

// sendAs records one plugin message of the plugin for the person, and answers its id.
func sendAs(
	t *testing.T,
	instance *testdb.Instance,
	user string,
	message model.TelegramMessage,
) string {
	t.Helper()

	var id string

	require.NoError(t, asUser(t, instance, user, func(ctx context.Context, tx *sqlx.Tx) error {
		var err error

		id, err = repository.NewTelegramMessage(tx).Insert(ctx, &message)

		return err //nolint:wrapcheck // the test reads the error that the repository answered.
	}))

	return id
}

func messageOf(
	pluginID string,
	chatID, messageID int64,
	claimUntil *time.Time,
) model.TelegramMessage {
	return model.TelegramMessage{
		PluginID:   pluginID,
		ChatID:     chatID,
		MessageID:  messageID,
		ClaimUntil: claimUntil,
	}
}

// ageRows moves every row of the person back by the given age.
func ageRows(t *testing.T, instance *testdb.Instance, user string, age time.Duration) {
	t.Helper()

	require.NoError(t, asUser(t, instance, user, func(ctx context.Context, tx *sqlx.Tx) error {
		_, err := tx.ExecContext(
			ctx,
			`UPDATE public.telegram_messages SET created_at = created_at - make_interval(secs => $1);`,
			age.Seconds(),
		)

		return err //nolint:wrapcheck // the test reads the error that the database answered.
	}))
}

func readAs[T any](
	t *testing.T,
	instance *testdb.Instance,
	user string,
	read func(context.Context, *repository.TelegramMessage) (T, error),
) T {
	t.Helper()

	var result T

	require.NoError(t, asUser(t, instance, user, func(ctx context.Context, tx *sqlx.Tx) error {
		var err error

		result, err = read(ctx, repository.NewTelegramMessage(tx))

		return err
	}))

	return result
}

// TGMSG-INV-002: A press of Beka with the chat and the message of a plugin message of
// Nina finds no row, because the policy hides it.
func TestTheRowOfAPluginMessageIsHiddenFromAnotherPerson(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	nina := insertUser(t, instance, nameOfA)
	beka := insertUser(t, instance, nameOfB)
	id := sendAs(t, instance, nina, messageOf(pluginOfMessages, chatOfNina, 10, nil))

	byTelegram := func(ctx context.Context, messages *repository.TelegramMessage) (*model.TelegramMessage, error) {
		return messages.ByTelegram(ctx, chatOfNina, 10)
	}

	own := readAs(t, instance, nina, byTelegram)
	require.NotNil(t, own)
	assert.Equal(t, id, own.ID)
	assert.Equal(t, pluginOfMessages, own.PluginID)

	assert.Nil(t, readAs(t, instance, beka, byTelegram))
}

// TGMSG-SC-008: A plugin reads its own message alone, and no message older than 30 days.
func TestAPluginReadsItsOwnMessageForThirtyDays(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	nina := insertUser(t, instance, nameOfA)
	id := sendAs(t, instance, nina, messageOf(pluginOfMessages, chatOfNina, 11, nil))

	byID := func(pluginID string) func(context.Context, *repository.TelegramMessage) (*model.TelegramMessage, error) {
		return func(ctx context.Context, messages *repository.TelegramMessage) (*model.TelegramMessage, error) {
			return messages.ByID(ctx, id, pluginID)
		}
	}

	require.NotNil(t, readAs(t, instance, nina, byID(pluginOfMessages)))
	assert.Nil(t, readAs(t, instance, nina, byID(pluginOfQ)), "Q sent no such message")

	ageRows(t, instance, nina, lifetimeAndADay)

	assert.Nil(t, readAs(t, instance, nina, byID(pluginOfMessages)), "day 31 is past the lifetime")
	assert.Nil(t, readAs(
		t,
		instance,
		nina,
		func(ctx context.Context, messages *repository.TelegramMessage) (*model.TelegramMessage, error) {
			return messages.ByTelegram(ctx, chatOfNina, 11)
		},
	))
}

// TGMSG-SC-007: The record refuses a claim that ends after the message stops working.
func TestTheRecordRefusesAClaimPastThirtyDays(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	nina := insertUser(t, instance, nameOfA)
	tooLate := time.Now().Add(lifetimeAndADay)

	err := asUser(t, instance, nina, func(ctx context.Context, tx *sqlx.Tx) error {
		message := messageOf(pluginOfMessages, chatOfNina, 12, &tooLate)
		_, insertErr := repository.NewTelegramMessage(tx).Insert(ctx, &message)

		return insertErr //nolint:wrapcheck // the test reads the error that the repository answered.
	})

	require.ErrorContains(t, err, "telegram_messages_claim_check")
}

// TGMSG-SC-016: The open claims come oldest first, and a released or expired claim
// is not open.
func TestTheOpenClaimsComeOldestFirst(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	nina := insertUser(t, instance, nameOfA)
	later := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Minute)

	first := sendAs(t, instance, nina, messageOf(pluginOfMessages, chatOfNina, 20, &later))
	second := sendAs(t, instance, nina, messageOf(pluginOfQ, chatOfNina, 21, &later))
	sendAs(t, instance, nina, messageOf(pluginOfMessages, chatOfNina, 22, &past))
	sendAs(t, instance, nina, messageOf(pluginOfMessages, chatOfNina, 23, nil))

	openClaims := func(ctx context.Context, messages *repository.TelegramMessage) ([]model.TelegramMessage, error) {
		return messages.OpenClaims(ctx)
	}

	claims := readAs(t, instance, nina, openClaims)
	require.Len(t, claims, 2)
	assert.Equal(t, first, claims[0].ID)
	assert.Equal(t, second, claims[1].ID)
	assert.True(t, claims[0].ClaimOpen)

	expired := readAs(
		t,
		instance,
		nina,
		func(ctx context.Context, messages *repository.TelegramMessage) (*model.TelegramMessage, error) {
			return messages.ByTelegram(ctx, chatOfNina, 22)
		},
	)
	require.NotNil(t, expired)
	assert.False(t, expired.ClaimOpen, "a claim past its expiry is not open")
}

// TGMSG-SC-014, TGMSG-FR-014: A release ends the claim of the plugin that holds it,
// and a second release finds the row again.
func TestAReleaseEndsTheClaim(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	nina := insertUser(t, instance, nameOfA)
	later := time.Now().Add(time.Hour)
	id := sendAs(t, instance, nina, messageOf(pluginOfMessages, chatOfNina, 30, &later))

	release := func(pluginID string) func(context.Context, *repository.TelegramMessage) (bool, error) {
		return func(ctx context.Context, messages *repository.TelegramMessage) (bool, error) {
			return messages.Release(ctx, id, pluginID)
		}
	}

	assert.False(t, readAs(t, instance, nina, release(pluginOfQ)), "Q holds no such claim")
	assert.True(t, readAs(t, instance, nina, release(pluginOfMessages)))
	assert.True(
		t,
		readAs(t, instance, nina, release(pluginOfMessages)),
		"a second release finds the row",
	)

	claims := readAs(
		t,
		instance,
		nina,
		func(ctx context.Context, messages *repository.TelegramMessage) ([]model.TelegramMessage, error) {
			return messages.OpenClaims(ctx)
		},
	)
	assert.Empty(t, claims)
}

// TGMSG-SC-019: A purge deletes the rows of the acting user older than 30 days, and
// keeps the rows of another person.
func TestAPurgeDeletesTheOldRowsOfThePerson(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	nina := insertUser(t, instance, nameOfA)
	beka := insertUser(t, instance, nameOfB)

	sendAs(t, instance, nina, messageOf(pluginOfMessages, chatOfNina, 40, nil))
	sendAs(t, instance, beka, messageOf(pluginOfMessages, chatOfBeka, 41, nil))
	ageRows(t, instance, nina, lifetimeAndADay)
	ageRows(t, instance, beka, lifetimeAndADay)
	fresh := sendAs(t, instance, nina, messageOf(pluginOfMessages, chatOfNina, 42, nil))

	require.NoError(t, asUser(t, instance, nina, func(ctx context.Context, tx *sqlx.Tx) error {
		return repository.NewTelegramMessage(tx).Purge(ctx)
	}))

	countAs := func(user string) int {
		var count int

		require.NoError(t, asUser(t, instance, user, func(ctx context.Context, tx *sqlx.Tx) error {
			return tx.GetContext(ctx, &count, `SELECT count(*) FROM public.telegram_messages;`)
		}))

		return count
	}

	assert.Equal(t, 1, countAs(nina), "the fresh row stays")
	assert.Equal(t, 1, countAs(beka), "the old row of Beka stays")
	assert.NotEmpty(t, fresh)
}
