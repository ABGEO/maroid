package auth_test

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authtest"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

const (
	invitationTTL = 72 * time.Hour
	nameOfInvitee = "Nino"
)

// EXTID-SC-012: One action creates the user record and the invitation for it.
// EXTID-FR-010: The owner cannot write the first identity, because the identifier
// of an external account exists only after that person signs in one time.
func TestInviteCreatesTheRecordAndTheGrant(t *testing.T) {
	t.Parallel()

	service, _, database := serviceUnderTest(t)
	ctx := t.Context()
	invitationRepo := repository.NewInvitation(database)

	result, err := service.Invite(ctx, auth.InviteRequest{
		FirstName: nameOfInvitee,
		LastName:  "Beridze",
	}, invitationTTL)
	require.NoError(t, err)
	require.NotEmpty(t, result.UserID)
	require.NotEmpty(t, result.Token)

	var person struct {
		FirstName *string `db:"first_name"`
		LastName  *string `db:"last_name"`
	}

	require.NoError(t, database.Get(&person,
		`SELECT first_name, last_name FROM public.users WHERE id = $1;`, result.UserID))
	require.Equal(t, nameOfInvitee, *person.FirstName)
	require.Equal(t, "Beridze", *person.LastName)

	digest := sha256.Sum256([]byte(result.Token))

	invitation, err := invitationRepo.GetValidByTokenHash(ctx, digest[:])
	require.NoError(t, err)
	require.Equal(t, result.UserID, invitation.UserID)

	// EXTID-DD-007: The row holds the digest, so a reader of the database gains
	// no capability.
	var stored []byte

	require.NoError(t, database.Get(&stored, `SELECT token_hash FROM public.invitations;`))
	require.NotContains(t, string(stored), result.Token)
}

// EXTID-SC-013: The owner issues a second invitation for a record that exists,
// and the count of the records does not change.
func TestInviteForARecordThatExists(t *testing.T) {
	t.Parallel()

	service, _, database := serviceUnderTest(t)
	ctx := t.Context()

	userID := addUser(t, database, "Temuri")

	result, err := service.Invite(ctx, auth.InviteRequest{UserID: userID}, invitationTTL)
	require.NoError(t, err)
	require.Equal(t, userID, result.UserID)

	var count int

	require.NoError(t, database.Get(&count, `SELECT count(*) FROM public.users;`))
	require.Equal(t, 1, count, "no second record")

	_, err = service.Invite(ctx, auth.InviteRequest{
		UserID: "01998aa0-0000-7000-8000-000000000000",
	}, invitationTTL)
	require.Error(t, err, "an invitation needs a record that exists")
}

// EXTID-SC-014: The redemption spends the invitation and writes the first
// identity of its record, in one transaction. EXTID-DD-008 gives the order.
func TestRedeemBindsTheFirstIdentity(t *testing.T) {
	t.Parallel()

	service, identityRepo, database := serviceUnderTest(t)
	ctx := t.Context()
	invitationRepo := repository.NewInvitation(database)

	result, err := service.Invite(ctx, auth.InviteRequest{FirstName: nameOfInvitee}, invitationTTL)
	require.NoError(t, err)

	digest := sha256.Sum256([]byte(result.Token))

	invitation, err := invitationRepo.GetValidByTokenHash(ctx, digest[:])
	require.NoError(t, err)

	user, err := service.Redeem(ctx, invitation.ID, providerCloud, "abc", model.Profile{
		Username: authtest.HandleOfA,
	})
	require.NoError(t, err)
	require.Equal(t, result.UserID, user.ID)

	identities, err := identityRepo.ListByUser(ctx, result.UserID)
	require.NoError(t, err)
	require.Len(t, identities, 1)
	require.Equal(t, providerCloud, identities[0].Provider)

	_, err = invitationRepo.GetValidByTokenHash(ctx, digest[:])
	require.ErrorIs(t, err, errs.ErrInvitationNotValid, "the grant is spent")
}

// EXTID-SC-015: One invitation serves one time. A second person who holds the
// same address reaches nothing.
func TestRedeemSpendsTheInvitationOneTime(t *testing.T) {
	t.Parallel()

	service, identityRepo, database := serviceUnderTest(t)
	ctx := t.Context()
	invitationRepo := repository.NewInvitation(database)

	result, err := service.Invite(ctx, auth.InviteRequest{FirstName: nameOfInvitee}, invitationTTL)
	require.NoError(t, err)

	digest := sha256.Sum256([]byte(result.Token))

	invitation, err := invitationRepo.GetValidByTokenHash(ctx, digest[:])
	require.NoError(t, err)

	_, err = service.Redeem(ctx, invitation.ID, providerCloud, "abc", model.Profile{})
	require.NoError(t, err)

	_, err = service.Redeem(ctx, invitation.ID, auth.ProviderTelegram, "111", model.Profile{})
	require.ErrorIs(t, err, errs.ErrInvitationNotValid)

	identities, err := identityRepo.ListByUser(ctx, result.UserID)
	require.NoError(t, err)
	require.Len(t, identities, 1, "the second attempt wrote nothing")
}

// EXTID-SC-016: An invitation grants nothing after its validity ends.
func TestRedeemRefusesAnExpiredInvitation(t *testing.T) {
	t.Parallel()

	service, identityRepo, database := serviceUnderTest(t)
	ctx := t.Context()

	result, err := service.Invite(ctx, auth.InviteRequest{FirstName: nameOfInvitee}, -time.Minute)
	require.NoError(t, err)

	var invitationID string

	require.NoError(t, database.Get(&invitationID, `SELECT id FROM public.invitations;`))

	_, err = service.Redeem(ctx, invitationID, providerCloud, "abc", model.Profile{})
	require.ErrorIs(t, err, errs.ErrInvitationNotValid)

	identities, err := identityRepo.ListByUser(ctx, result.UserID)
	require.NoError(t, err)
	require.Empty(t, identities)
}

// EXTID-DD-008: A conflict does not burn the grant. The identity goes in the same
// transaction, so a failed insert returns the invitation to its unspent state.
func TestRedeemKeepsTheGrantWhenTheAccountIsTaken(t *testing.T) {
	t.Parallel()

	service, _, database := serviceUnderTest(t)
	ctx := t.Context()
	invitationRepo := repository.NewInvitation(database)

	holder := addUser(t, database, "Temuri")
	require.NoError(t, service.Attach(ctx, holder, providerCloud, "abc", model.Profile{}))

	result, err := service.Invite(ctx, auth.InviteRequest{FirstName: nameOfInvitee}, invitationTTL)
	require.NoError(t, err)

	digest := sha256.Sum256([]byte(result.Token))

	invitation, err := invitationRepo.GetValidByTokenHash(ctx, digest[:])
	require.NoError(t, err)

	_, err = service.Redeem(ctx, invitation.ID, providerCloud, "abc", model.Profile{})
	require.ErrorIs(t, err, errs.ErrIdentityTaken)

	_, err = invitationRepo.GetValidByTokenHash(ctx, digest[:])
	require.NoError(t, err, "the grant survives a conflict")
}
