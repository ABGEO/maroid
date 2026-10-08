package repository_test

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
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
	userID string,
	token string,
	expiresAt time.Time,
) (*model.Invitation, []byte) {
	t.Helper()

	digest := sha256.Sum256([]byte(token))

	invitation, err := fetch(t, instance, func(tx *sqlx.Tx) (*model.Invitation, error) {
		return repository.NewInvitation(tx).Create(t.Context(), userID, digest[:], expiresAt)
	})
	require.NoError(t, err)

	return invitation, digest[:]
}

func validInvitation(
	t *testing.T,
	instance *testdb.Instance,
	digest []byte,
) (*model.Invitation, error) {
	t.Helper()

	return fetch(t, instance, func(tx *sqlx.Tx) (*model.Invitation, error) {
		return repository.NewInvitation(tx).GetValidByTokenHash(t.Context(), digest)
	})
}

func consumeInvitation(
	t *testing.T,
	instance *testdb.Instance,
	id string,
) (*model.Invitation, error) {
	t.Helper()

	return fetch(t, instance, func(tx *sqlx.Tx) (*model.Invitation, error) {
		return repository.NewInvitation(tx).Consume(t.Context(), id)
	})
}

// EXTID-FR-013: One invitation serves one time.
// EXTID-DD-008: The condition on consumed_at is the gate, so exactly one caller
// sees one affected row.
func TestInvitationConsumesOneTime(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)

	userID := insertUser(t, instance, nameOfA)
	created, digest := issue(t, instance, userID, "the-token", time.Now().Add(invitationTTL))

	require.Nil(t, created.ConsumedAt)
	require.Equal(t, userID, created.UserID)

	found, err := validInvitation(t, instance, digest)
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID)

	spent, err := consumeInvitation(t, instance, created.ID)
	require.NoError(t, err)
	require.Equal(t, userID, spent.UserID, "the row names the record it grants")

	_, err = consumeInvitation(t, instance, created.ID)
	require.ErrorIs(t, err, errs.ErrInvitationNotValid)

	_, err = validInvitation(t, instance, digest)
	require.ErrorIs(t, err, errs.ErrInvitationNotValid, "a consumed invitation grants nothing")
}

// EXTID-FR-014: An invitation grants nothing after its validity ends.
func TestExpiredInvitationGrantsNothing(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)

	userID := insertUser(t, instance, nameOfA)
	created, digest := issue(t, instance, userID, "stale", time.Now().Add(-time.Minute))

	_, err := validInvitation(t, instance, digest)
	require.ErrorIs(t, err, errs.ErrInvitationNotValid)

	_, err = consumeInvitation(t, instance, created.ID)
	require.ErrorIs(t, err, errs.ErrInvitationNotValid)
}

// EXTID-DD-007: The column holds the digest, so a reader of the database gains no
// capability. An absent digest answers as a consumed one.
func TestInvitationLookupNeedsTheDigest(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)

	userID := insertUser(t, instance, nameOfA)
	_, digest := issue(t, instance, userID, "the-token", time.Now().Add(invitationTTL))

	var stored []byte

	require.NoError(t, instance.DB.Get(&stored, `SELECT token_hash FROM public.invitations;`))
	require.Equal(t, digest, stored)
	require.NotContains(t, string(stored), "the-token", "the token never rests in the row")

	other := sha256.Sum256([]byte("another-token"))

	_, err := validInvitation(t, instance, other[:])
	require.ErrorIs(t, err, errs.ErrInvitationNotValid)
}
