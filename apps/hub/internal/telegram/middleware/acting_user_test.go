//nolint:testpackage // resolveActingUser is unexported, and it holds the decision.
package middleware

import (
	"context"
	"log/slog"
	"testing"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
)

const (
	recordID   = "01998aa0-1111-7000-8000-00000000000a"
	telegramID = int64(722183546)
)

// fakeResolver answers with the record that the test gives, or with the error,
// and it reports what the middleware asked for.
type fakeResolver struct {
	user           *model.User
	err            error
	gotProvider    string
	gotProviderUID string
}

var _ auth.IdentityResolver = (*fakeResolver)(nil)

func (f *fakeResolver) ResolveByProvider(
	_ context.Context,
	provider string,
	providerUserID string,
) (*model.User, error) {
	f.gotProvider = provider
	f.gotProviderUID = providerUserID

	return f.user, f.err
}

func updateFrom(id int64) telego.Update {
	return telego.Update{
		Message: &telego.Message{
			From: &telego.User{ID: id, Username: "abgeo"},
		},
	}
}

// IDENT-FR-002: Maroid refuses every interaction from a person that holds no
// active user record, in the bot as in the web shell.
// EXTID-FR-001: An external account that holds no identity reaches nothing.
func TestAnUpdateFromAPersonWithNoActiveRecordIsDropped(t *testing.T) {
	t.Parallel()

	acting, ok := resolveActingUser(
		t.Context(),
		slog.New(slog.DiscardHandler),
		&fakeResolver{err: errs.ErrUserNotFound},
		updateFrom(telegramID),
	)

	require.False(t, ok, "the update must reach no handler")
	require.Empty(t, acting)
}

// EXTID-FR-017: The bot resolves through the identity, so it reaches the record
// that the web shell reaches. The numeric sender is the external account, which
// is why ADR-0002 binds userIDKey on the connector.
func TestAnUpdateCarriesTheActingUser(t *testing.T) {
	t.Parallel()

	resolver := &fakeResolver{
		user: &model.User{ID: recordID, Status: model.StatusActive},
	}

	acting, ok := resolveActingUser(
		t.Context(),
		slog.New(slog.DiscardHandler),
		resolver,
		updateFrom(telegramID),
	)

	require.True(t, ok)
	require.Equal(t, recordID, acting)
	require.Equal(t, auth.ProviderTelegram, resolver.gotProvider)
	require.Equal(t, "722183546", resolver.gotProviderUID, "the sender is the external account")
}

// An update that names no sender reaches nothing, because no record owns it.
func TestAnUpdateWithNoSenderIsDropped(t *testing.T) {
	t.Parallel()

	resolver := &fakeResolver{}

	acting, ok := resolveActingUser(
		t.Context(),
		slog.New(slog.DiscardHandler),
		resolver,
		telego.Update{Message: nil},
	)

	require.False(t, ok)
	require.Empty(t, acting)
	require.Empty(t, resolver.gotProvider, "no sender means no resolution")
}
