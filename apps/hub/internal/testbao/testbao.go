// Package testbao starts an OpenBao server that one integration test owns, with the
// transit engine and the AppRole login that Maroid uses.
package testbao

import (
	"context"
	"log/slog"
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
	// TransitMount is the path of the transit engine.
	TransitMount = "transit"

	image        = "quay.io/openbao/openbao:2.5"
	rootToken    = "root-token"
	startTimeout = 2 * time.Minute
	role         = "maroid"
)

const policy = `
path "transit/encrypt/maroid-user-*" { capabilities = ["update"] }
path "transit/decrypt/maroid-user-*" { capabilities = ["update"] }
path "transit/encrypt/maroid-workspace-*" { capabilities = ["update"] }
path "transit/decrypt/maroid-workspace-*" { capabilities = ["update"] }
`

// Server is a running OpenBao that one test owns.
type Server struct {
	// Root holds the root token. The test creates the keys with it.
	Root *api.Client
}

// Start launches OpenBao in a container, mounts the transit engine, and creates the
// role that Maroid logs in with.
func Start(t *testing.T) *Server {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
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

	clientConfig := api.DefaultConfig()
	clientConfig.Address = endpoint

	root, err := api.NewClient(clientConfig)
	require.NoError(t, err)
	root.SetToken(rootToken)

	require.NoError(t, root.Sys().MountWithContext(
		t.Context(), TransitMount, &api.MountInput{Type: "transit"},
	))
	require.NoError(t, root.Sys().EnableAuthWithOptionsWithContext(
		t.Context(), "approle", &api.EnableAuthOptions{Type: "approle"},
	))
	require.NoError(t, root.Sys().PutPolicyWithContext(t.Context(), role, policy))

	_, err = root.Logical().WriteWithContext(
		t.Context(), "auth/approle/role/"+role, map[string]any{"token_policies": role},
	)
	require.NoError(t, err)

	return &Server{Root: root}
}

// CreateKeys creates one transit key for each name, as the owner does.
func (s *Server) CreateKeys(t *testing.T, keys ...secret.Key) {
	t.Helper()

	for _, key := range keys {
		_, err := s.Root.Logical().WriteWithContext(t.Context(), "transit/keys/"+string(key), nil)
		require.NoError(t, err)
	}
}

// Cipher logs in as Maroid does, and returns the cipher over that session.
func (s *Server) Cipher(t *testing.T) *secret.Transit {
	t.Helper()

	session, err := openbao.New(t.Context(), &config.OpenBao{
		Address:      s.Root.Address(),
		RoleID:       s.readRoleID(t),
		SecretID:     s.generateSecretID(t),
		TransitMount: TransitMount,
	}, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	return secret.NewTransit(session.Client(), TransitMount)
}

func (s *Server) readRoleID(t *testing.T) string {
	t.Helper()

	answer, err := s.Root.Logical().
		ReadWithContext(t.Context(), "auth/approle/role/"+role+"/role-id")
	require.NoError(t, err)

	value, ok := answer.Data["role_id"].(string)
	require.True(t, ok)

	return value
}

func (s *Server) generateSecretID(t *testing.T) string {
	t.Helper()

	answer, err := s.Root.Logical().WriteWithContext(
		t.Context(), "auth/approle/role/"+role+"/secret-id", nil,
	)
	require.NoError(t, err)

	value, ok := answer.Data["secret_id"].(string)
	require.True(t, ok)

	return value
}
