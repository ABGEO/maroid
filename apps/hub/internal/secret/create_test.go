package secret_test

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/openbao/openbao/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/openbao"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/apps/hub/internal/testbao"
)

const absentWorkspace = "55555555-5555-5555-5555-555555555555"

func requireDenied(t *testing.T, err error) {
	t.Helper()

	var answer *api.ResponseError

	require.ErrorAs(t, err, &answer)
	assert.Equal(t, http.StatusForbidden, answer.StatusCode)
}

// OWNKEY-SC-016: The role of the hub creates no key outside the two prefixes, creates
// no key through an encrypt, and reads no export of a key.
func TestTheHubMintsNoOtherKey(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	server := testbao.Start(t)

	session, err := openbao.New(t.Context(), server.Config(t), slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	hub := session.Client().Logical()
	absent := secret.WorkspaceKey(absentWorkspace)

	_, err = hub.WriteWithContext(t.Context(), testbao.TransitMount+"/keys/other-key", nil)
	requireDenied(t, err)

	_, err = hub.WriteWithContext(
		t.Context(),
		testbao.TransitMount+"/encrypt/"+string(absent),
		map[string]any{"plaintext": "aGk="},
	)
	requireDenied(t, err)

	_, err = hub.ReadWithContext(
		t.Context(),
		testbao.TransitMount+"/export/encryption-key/"+string(secret.UserKey(userA)),
	)
	requireDenied(t, err)

	for _, name := range []string{"other-key", string(absent)} {
		held, readErr := server.Root.Logical().
			ReadWithContext(t.Context(), testbao.TransitMount+"/keys/"+name)
		require.NoError(t, readErr)
		assert.Nil(t, held, name)
	}
}
