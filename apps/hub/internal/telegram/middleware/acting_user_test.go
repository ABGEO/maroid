//nolint:testpackage // resolveActingUser is unexported, and it holds the decision.
package middleware

import (
	"context"
	"log/slog"
	"testing"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

const recordID = "01998aa0-1111-7000-8000-00000000000a"

// fakeUserRepo answers with the record that the test gives, or with the error.
// The methods that the test does not reach report a missing record.
type fakeUserRepo struct {
	user *model.User
	err  error
}

var _ repository.UserRepository = (*fakeUserRepo)(nil)

func (f fakeUserRepo) GetActiveByID(context.Context, string) (*model.User, error) {
	return f.user, f.err
}

func (f fakeUserRepo) GetActiveByTelegramID(context.Context, int64) (*model.User, error) {
	return f.user, f.err
}

func (f fakeUserRepo) ListActive(context.Context) ([]model.User, error) {
	return nil, errs.ErrUserNotFound
}

func (f fakeUserRepo) SyncProfileByTelegramID(
	context.Context, int64, model.Profile,
) (*model.User, error) {
	return f.user, f.err
}

func updateFrom(telegramID int64) telego.Update {
	return telego.Update{
		Message: &telego.Message{
			From: &telego.User{ID: telegramID, Username: "abgeo"},
		},
	}
}

// IDENT-FR-002: Maroid refuses every interaction from a person that holds no
// active user record, in the bot as in the web shell.
// IDENT-SC-007: A blocked person sends an update, and the bot drops it.
func TestAnUpdateFromAPersonWithNoActiveRecordIsDropped(t *testing.T) {
	t.Parallel()

	acting, ok := resolveActingUser(
		t.Context(),
		slog.New(slog.DiscardHandler),
		fakeUserRepo{user: nil, err: errs.ErrUserNotFound},
		updateFrom(722183546),
	)

	require.False(t, ok, "the update must reach no handler")
	require.Empty(t, acting)
}

// OWN-003: The Telegram user identifier of the update names the acting user.
func TestAnUpdateCarriesTheActingUser(t *testing.T) {
	t.Parallel()

	acting, ok := resolveActingUser(
		t.Context(),
		slog.New(slog.DiscardHandler),
		fakeUserRepo{
			user: &model.User{ID: recordID, TelegramID: 722183546, Status: model.StatusActive},
			err:  nil,
		},
		updateFrom(722183546),
	)

	require.True(t, ok)
	require.Equal(t, recordID, acting)
}

// An update that names no sender reaches nothing, because no record owns it.
func TestAnUpdateWithNoSenderIsDropped(t *testing.T) {
	t.Parallel()

	acting, ok := resolveActingUser(
		t.Context(),
		slog.New(slog.DiscardHandler),
		fakeUserRepo{user: nil, err: nil},
		telego.Update{Message: nil},
	)

	require.False(t, ok)
	require.Empty(t, acting)
}
