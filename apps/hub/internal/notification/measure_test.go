//go:build measure

package notification_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/openbao/openbao/api/v2"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/abgeo/maroid/apps/hub/internal/secret"
)

const (
	baoImage         = "quay.io/openbao/openbao:2.5"
	baoRootToken     = "root-token"
	baoStartTimeout  = 2 * time.Minute
	measuredSends    = 200
	measuredRoutes   = 3
	measuredLimit    = 50 * time.Millisecond
	percentileNinety = 95
)

// startTransit starts OpenBao in development mode with a transit engine, and answers
// a client that holds the root token.
func startTransit(t *testing.T) *api.Client {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), baoStartTimeout)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        baoImage,
			ExposedPorts: []string{"8200/tcp"},
			Env: map[string]string{
				"BAO_DEV_ROOT_TOKEN_ID":  baoRootToken,
				"BAO_DEV_LISTEN_ADDRESS": "0.0.0.0:8200",
			},
			WaitingFor: wait.ForHTTP("/v1/sys/health").
				WithPort("8200/tcp").
				WithStatusCodeMatcher(func(status int) bool { return status == http.StatusOK }).
				WithStartupTimeout(baoStartTimeout),
		},
		Started: true,
	})
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, container)

	endpoint, err := container.PortEndpoint(ctx, "8200/tcp", "http")
	require.NoError(t, err)

	clientConfig := api.DefaultConfig()
	clientConfig.Address = endpoint

	client, err := api.NewClient(clientConfig)
	require.NoError(t, err)
	client.SetToken(baoRootToken)

	require.NoError(
		t,
		client.Sys().MountWithContext(t.Context(), "transit", &api.MountInput{Type: "transit"}),
	)

	return client
}

// NOTIF-SC-027: With 10 providers and 50 routes, of which 3 serve the topic, the time
// from the send to the first call at Telegram stays within 50 milliseconds at the
// 95th percentile.
//
//nolint:paralleltest // a measurement runs alone, so no other test takes its time
func TestTheRoutesOfOneSendResolveWithinFiftyMilliseconds(t *testing.T) {
	bao := startTransit(t)
	world := newDeliveryWorldWith(t, secret.NewTransit(bao, "transit"))

	_, err := bao.Logical().WriteWithContext(
		t.Context(), "transit/keys/"+string(secret.WorkspaceKey(world.workspaceA)), nil,
	)
	require.NoError(t, err)

	providers := make([]string, 0, 10)
	for index := range 10 {
		providers = append(
			providers,
			world.addProvider(t, world.workspaceA, fmt.Sprintf("Bot %d", index)),
		)
	}

	for index := range 50 {
		topic := reportTopic
		if index < measuredRoutes {
			topic = reminderTopic
		}

		world.addRoute(
			t,
			world.workspaceA,
			providers[index%len(providers)],
			topic,
			fmt.Sprintf("-%d", 100+index),
		)
	}

	durations := make([]time.Duration, 0, measuredSends)

	for range measuredSends {
		before := len(world.stub.Calls())
		start := time.Now()

		require.NoError(t, world.send(world.workspaceA, reminderTopic))

		first := world.stub.Calls()[before]
		durations = append(durations, first.At.Sub(start))
	}

	slices.Sort(durations)
	percentile := durations[len(durations)*percentileNinety/100-1]

	t.Logf("the 95th percentile from the send to the first call is %s", percentile)
	require.LessOrEqual(t, percentile, measuredLimit)
}
