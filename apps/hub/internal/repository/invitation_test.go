package repository_test

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

const invitationTTL = 72 * time.Hour

// issue writes one invitation and returns it with the digest that names it.
func issue(
	t *testing.T,
	instance *testdb.Instance,
	repo repository.InvitationRepository,
	userID string,
	token string,
	expiresAt time.Time,
) (*model.Invitation, []byte) {
	t.Helper()

	digest := sha256.Sum256([]byte(token))

	tx, err := instance.DB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	invitation, err := repo.Create(t.Context(), tx, userID, digest[:], expiresAt)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	return invitation, digest[:]
}

// EXTID-FR-013: One invitation serves one time.
// EXTID-DD-008: The condition on consumed_at is the gate, so exactly one caller
// sees one affected row.
func TestInvitationConsumesOneTime(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	invitationRepo := repository.NewInvitation(instance.DB)
	ctx := t.Context()

	userID := insertUser(t, instance, telegramIDOfA)
	created, digest := issue(
		t, instance, invitationRepo, userID, "the-token", time.Now().Add(invitationTTL),
	)

	require.Nil(t, created.ConsumedAt)
	require.Equal(t, userID, created.UserID)

	found, err := invitationRepo.GetValidByTokenHash(ctx, digest)
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID)

	tx, err := instance.DB.BeginTxx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, invitationRepo.Consume(ctx, tx, created.ID))
	require.NoError(t, tx.Commit())

	tx, err = instance.DB.BeginTxx(ctx, nil)
	require.NoError(t, err)
	require.ErrorIs(t, invitationRepo.Consume(ctx, tx, created.ID), errs.ErrInvitationNotValid)
	require.NoError(t, tx.Rollback())

	_, err = invitationRepo.GetValidByTokenHash(ctx, digest)
	require.ErrorIs(t, err, errs.ErrInvitationNotValid, "a consumed invitation grants nothing")
}

// EXTID-FR-014: An invitation grants nothing after its validity ends.
func TestExpiredInvitationGrantsNothing(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	invitationRepo := repository.NewInvitation(instance.DB)
	ctx := t.Context()

	userID := insertUser(t, instance, telegramIDOfA)
	created, digest := issue(
		t, instance, invitationRepo, userID, "stale", time.Now().Add(-time.Minute),
	)

	_, err := invitationRepo.GetValidByTokenHash(ctx, digest)
	require.ErrorIs(t, err, errs.ErrInvitationNotValid)

	tx, err := instance.DB.BeginTxx(ctx, nil)
	require.NoError(t, err)
	require.ErrorIs(t, invitationRepo.Consume(ctx, tx, created.ID), errs.ErrInvitationNotValid)
	require.NoError(t, tx.Rollback())
}

// EXTID-DD-007: The column holds the digest, so a reader of the database gains no
// capability. An absent digest answers as a consumed one.
func TestInvitationLookupNeedsTheDigest(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	invitationRepo := repository.NewInvitation(instance.DB)

	userID := insertUser(t, instance, telegramIDOfA)
	_, digest := issue(
		t, instance, invitationRepo, userID, "the-token", time.Now().Add(invitationTTL),
	)

	var stored []byte

	require.NoError(t, instance.DB.Get(&stored, `SELECT token_hash FROM public.invitations;`))
	require.Equal(t, digest, stored)
	require.NotContains(t, string(stored), "the-token", "the token never rests in the row")

	other := sha256.Sum256([]byte("another-token"))

	_, err := invitationRepo.GetValidByTokenHash(t.Context(), other[:])
	require.ErrorIs(t, err, errs.ErrInvitationNotValid)
}
