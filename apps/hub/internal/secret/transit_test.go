package secret_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/openbao/openbao/api/v2"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/openbao"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
)

const (
	baoImage     = "quay.io/openbao/openbao:2.5"
	rootToken    = "root-token"
	startTimeout = 2 * time.Minute
	userA        = "11111111-1111-1111-1111-111111111111"
	userB        = "22222222-2222-2222-2222-222222222222"
)

// policy holds the capabilities that PSET-DD-004 gives the hub. It grants no create,
// because the owner creates every key by hand.
const policy = `
path "transit/encrypt/maroid-user-*" { capabilities = ["update"] }
path "transit/decrypt/maroid-user-*" { capabilities = ["update"] }
`

// startBao launches OpenBao in development mode, mounts the transit engine, creates
// one key for each user, and returns a cipher that logs in with AppRole.
func startBao(t *testing.T) (*secret.Transit, *api.Client) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        baoImage,
			ExposedPorts: []string{"8200/tcp"},
			Env: map[string]string{
				"BAO_DEV_ROOT_TOKEN_ID":  rootToken,
				"BAO_DEV_LISTEN_ADDRESS": "0.0.0.0:8200",
			},
			WaitingFor: wait.ForHTTP("/v1/sys/health").
				WithPort("8200/tcp").
				WithStatusCodeMatcher(func(status int) bool { return status == http.StatusOK }).
				WithStartupTimeout(startTimeout),
		},
		Started: true,
	})
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, container)

	endpoint, err := container.PortEndpoint(ctx, "8200/tcp", "http")
	require.NoError(t, err)

	root := rootClient(t, endpoint)
	prepare(t, root)

	client, err := openbao.New(t.Context(), &config.OpenBao{
		Address:      endpoint,
		RoleID:       readRoleID(t, root),
		SecretID:     generateSecretID(t, root),
		TransitMount: "transit",
	})
	require.NoError(t, err)

	cipher := secret.NewTransit(client, "transit")

	return cipher, root
}

func rootClient(t *testing.T, endpoint string) *api.Client {
	t.Helper()

	clientConfig := api.DefaultConfig()
	clientConfig.Address = endpoint

	client, err := api.NewClient(clientConfig)
	require.NoError(t, err)
	client.SetToken(rootToken)

	return client
}

// prepare does the work that the owner does by hand. See PSET-DD-004.
func prepare(t *testing.T, root *api.Client) {
	t.Helper()

	ctx := t.Context()

	require.NoError(
		t,
		root.Sys().MountWithContext(ctx, "transit", &api.MountInput{Type: "transit"}),
	)

	for _, user := range []string{userA, userB} {
		_, err := root.Logical().
			WriteWithContext(ctx, "transit/keys/"+string(secret.UserKey(user)), nil)
		require.NoError(t, err)
	}

	require.NoError(t, root.Sys().EnableAuthWithOptionsWithContext(
		ctx, "approle", &api.EnableAuthOptions{Type: "approle"},
	))
	require.NoError(t, root.Sys().PutPolicyWithContext(ctx, "maroid", policy))

	_, err := root.Logical().WriteWithContext(ctx, "auth/approle/role/maroid", map[string]any{
		"token_policies": "maroid",
	})
	require.NoError(t, err)
}

func readRoleID(t *testing.T, root *api.Client) string {
	t.Helper()

	answer, err := root.Logical().ReadWithContext(t.Context(), "auth/approle/role/maroid/role-id")
	require.NoError(t, err)

	return field(t, answer, "role_id")
}

func generateSecretID(t *testing.T, root *api.Client) string {
	t.Helper()

	answer, err := root.Logical().WriteWithContext(
		t.Context(), "auth/approle/role/maroid/secret-id", nil,
	)
	require.NoError(t, err)

	return field(t, answer, "secret_id")
}

func field(t *testing.T, answer *api.Secret, name string) string {
	t.Helper()

	value, ok := answer.Data[name].(string)
	require.True(t, ok, "the answer holds %s", name)

	return value
}

// PSET-SC-013: The protected form is not the plaintext.
// PSET-FR-018: startBao built the cipher, so the login already proved that the
// service answers and that the credentials work.
func TestEncryptReturnsAProtectedForm(t *testing.T) {
	t.Parallel()

	cipher, _ := startBao(t)

	ciphertext, err := cipher.Encrypt(t.Context(), secret.UserKey(userA), "hunter2")
	require.NoError(t, err)
	require.Contains(t, ciphertext, "vault:v1:")
	require.NotContains(t, ciphertext, "hunter2")

	plaintext, err := cipher.Decrypt(t.Context(), secret.UserKey(userA), ciphertext)
	require.NoError(t, err)
	require.Equal(t, "hunter2", plaintext)
}

// PSET-SC-014: A secret of user A does not decrypt under the key of user B.
func TestASecretDoesNotDecryptUnderTheKeyOfAnotherUser(t *testing.T) {
	t.Parallel()

	cipher, _ := startBao(t)

	ciphertext, err := cipher.Encrypt(t.Context(), secret.UserKey(userA), "hunter2")
	require.NoError(t, err)

	plaintext, err := cipher.Decrypt(t.Context(), secret.UserKey(userB), ciphertext)
	require.Error(t, err)
	require.Empty(t, plaintext)
}

// PSET-SC-015: A secret that version 1 protected reads after a rotation to version 2.
func TestASecretReadsAfterTheOwnerReplacedTheProtection(t *testing.T) {
	t.Parallel()

	cipher, root := startBao(t)

	ciphertext, err := cipher.Encrypt(t.Context(), secret.UserKey(userA), "hunter2")
	require.NoError(t, err)

	_, err = root.Logical().WriteWithContext(
		t.Context(), "transit/keys/"+string(secret.UserKey(userA))+"/rotate", nil,
	)
	require.NoError(t, err)

	rotated, err := cipher.Encrypt(t.Context(), secret.UserKey(userA), "hunter2")
	require.NoError(t, err)
	require.Contains(t, rotated, "vault:v2:")

	plaintext, err := cipher.Decrypt(t.Context(), secret.UserKey(userA), ciphertext)
	require.NoError(t, err)
	require.Equal(t, "hunter2", plaintext)
}
