package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
)

// PLUGACC-SC-001: An invitation with the mark of an administrator marks a new record
// and a record that exists, and each answers the token of an invitation.
func TestAnInvitationMarksAnAdministrator(t *testing.T) {
	t.Parallel()

	service, _, database := serviceUnderTest(t)
	ana := addUser(t, database, "Ana")

	nino, err := service.Invite(t.Context(), auth.InviteRequest{
		FirstName: nameOfInvitee, Administrator: true,
	}, invitationTTL)
	require.NoError(t, err)
	assert.NotEmpty(t, nino.Token)

	existing, err := service.Invite(t.Context(), auth.InviteRequest{
		UserID: ana, Administrator: true,
	}, invitationTTL)
	require.NoError(t, err)
	assert.NotEmpty(t, existing.Token)

	for _, id := range []string{nino.UserID, ana} {
		var marked bool

		require.NoError(t, database.Get(&marked,
			`SELECT is_administrator FROM public.users WHERE id = $1;`, id))
		assert.True(t, marked, "the record %s is an administrator", id)
	}
}

// PLUGACC-FR-001: An invitation with no mark leaves the record without one.
func TestAnInvitationWithNoMarkMarksNothing(t *testing.T) {
	t.Parallel()

	service, _, database := serviceUnderTest(t)

	invited, err := service.Invite(
		t.Context(),
		auth.InviteRequest{FirstName: nameOfInvitee},
		invitationTTL,
	)
	require.NoError(t, err)

	var marked bool

	require.NoError(t, database.Get(&marked,
		`SELECT is_administrator FROM public.users WHERE id = $1;`, invited.UserID))
	assert.False(t, marked)
}
