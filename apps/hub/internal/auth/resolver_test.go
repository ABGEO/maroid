package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
)

// EXTID-DD-012: One resolver serves both entry points, and it passes the provider
// and the external account through without a change.
func TestResolverReadsTheIdentity(t *testing.T) {
	t.Parallel()

	service, _, database := serviceUnderTest(t)
	userID := addUser(t, database, "Temuri")
	require.NoError(t, service.Attach(
		t.Context(), userID, auth.ProviderTelegram, "722183546", model.Profile{},
	))

	resolver := auth.NewResolver(database)

	user, err := resolver.ResolveByProvider(t.Context(), auth.ProviderTelegram, "722183546")
	require.NoError(t, err)
	require.Equal(t, userID, user.ID)

	_, err = resolver.ResolveByProvider(t.Context(), providerCloud, "722183546")
	require.ErrorIs(t, err, errs.ErrUserNotFound, "the provider takes part in the lookup")

	_, err = resolver.ResolveByProvider(t.Context(), auth.ProviderTelegram, "722183547")
	require.ErrorIs(t, err, errs.ErrUserNotFound, "the account takes part in the lookup")
}

// EXTID-FR-001: An external account that names no active record resolves to
// nothing, and the caller reads the reason.
func TestResolverReportsAnUnknownAccount(t *testing.T) {
	t.Parallel()

	_, _, database := serviceUnderTest(t)
	resolver := auth.NewResolver(database)

	user, err := resolver.ResolveByProvider(t.Context(), providerCloud, "nobody")
	require.ErrorIs(t, err, errs.ErrUserNotFound)
	require.Nil(t, user)
}
