package auth_test

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

// fakeIdentityRepository answers the resolution and reports what it was asked.
type fakeIdentityRepository struct {
	user           *model.User
	err            error
	gotProvider    string
	gotProviderUID string
}

var _ repository.IdentityRepository = (*fakeIdentityRepository)(nil)

func (f *fakeIdentityRepository) GetActiveUserByProvider(
	_ context.Context,
	provider string,
	providerUserID string,
) (*model.User, error) {
	f.gotProvider = provider
	f.gotProviderUID = providerUserID

	return f.user, f.err
}

func (f *fakeIdentityRepository) ListByUser(
	_ context.Context,
	_ string,
) ([]model.Identity, error) {
	return nil, errs.ErrIdentityNotFound
}

func (f *fakeIdentityRepository) Attach(
	_ context.Context,
	_ *sqlx.Tx,
	_ string,
	_ string,
	_ string,
	_ model.Profile,
) error {
	return errs.ErrIdentityNotFound
}

func (f *fakeIdentityRepository) SyncProfile(
	_ context.Context,
	_ string,
	_ string,
	_ model.Profile,
) error {
	return errs.ErrIdentityNotFound
}

func (f *fakeIdentityRepository) Detach(_ context.Context, _ string, _ string) error {
	return errs.ErrIdentityNotFound
}

// EXTID-DD-012: One resolver serves both entry points, and it passes the provider
// and the external account through without a change.
func TestResolverReadsTheIdentity(t *testing.T) {
	t.Parallel()

	expected := &model.User{ID: "01998aa0-1111-7000-8000-00000000000a"}
	identityRepo := &fakeIdentityRepository{user: expected}
	resolver := auth.NewResolver(identityRepo)

	user, err := resolver.ResolveByProvider(t.Context(), "telegram", "722183546")
	require.NoError(t, err)
	require.Equal(t, expected.ID, user.ID)
	require.Equal(t, "telegram", identityRepo.gotProvider)
	require.Equal(t, "722183546", identityRepo.gotProviderUID)
}

// EXTID-FR-001: An external account that names no active record resolves to
// nothing, and the caller reads the reason.
func TestResolverReportsAnUnknownAccount(t *testing.T) {
	t.Parallel()

	resolver := auth.NewResolver(&fakeIdentityRepository{err: errs.ErrUserNotFound})

	user, err := resolver.ResolveByProvider(t.Context(), "cloud", "nobody")
	require.ErrorIs(t, err, errs.ErrUserNotFound)
	require.Nil(t, user)
}
