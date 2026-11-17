// Package teststore starts a RustFS server that one integration test owns, with the
// bucket that the hub uses.
package teststore

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mcuadros/go-defaults"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

const (
	image        = "rustfs/rustfs:1.0.4"
	port         = "9000/tcp"
	startTimeout = 2 * time.Minute
	stopTimeout  = 10 * time.Second
	retryEvery   = 500 * time.Millisecond

	// Bucket is the bucket of the hub.
	Bucket = "maroid"
	// AccessKey names the key that signs every request of the test.
	AccessKey = "maroid-hub"
	// SecretKey is the secret of AccessKey.
	SecretKey = "maroid-hub-secret" //nolint:gosec // the key of a server that one test owns.
)

// Server is a running RustFS that one test owns.
type Server struct {
	// Endpoint is the address of the server on the host of the test.
	Endpoint string

	container testcontainers.Container
}

// Start launches RustFS in a container and creates the bucket of the hub.
func Start(t *testing.T) *Server {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
			ExposedPorts: []string{port},
			Env: map[string]string{
				"RUSTFS_ACCESS_KEY": AccessKey,
				"RUSTFS_SECRET_KEY": SecretKey,
			},
			WaitingFor: wait.ForHTTP("/health").
				WithPort(port).
				WithStatusCodeMatcher(func(status int) bool { return status == http.StatusOK }).
				WithStartupTimeout(startTimeout),
		},
		Started: true,
	})
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, container)

	endpoint, err := container.PortEndpoint(ctx, port, "http")
	require.NoError(t, err)

	server := &Server{Endpoint: endpoint, container: container}
	server.createBucket(ctx, t)

	return server
}

// Config answers the configuration of the files that reaches this server, with the
// default of each other key. The public endpoint is the same address.
func (s *Server) Config() config.Files {
	cfg := config.Files{
		Endpoint:       s.Endpoint,
		PublicEndpoint: s.Endpoint,
		Bucket:         Bucket,
		AccessKey:      AccessKey,
		SecretKey:      SecretKey,
	}
	defaults.SetDefaults(&cfg)

	return cfg
}

// Stop halts the server, so a test meets a file store that does not answer.
func (s *Server) Stop(t *testing.T) {
	t.Helper()

	timeout := stopTimeout
	require.NoError(t, s.container.Stop(context.Background(), &timeout))
}

func (s *Server) createBucket(ctx context.Context, t *testing.T) {
	t.Helper()

	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(s.Endpoint),
		UsePathStyle: true,
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider(AccessKey, SecretKey, ""),
	})

	// The first request after the start can meet a server that still opens its disks.
	require.Eventually(t, func() bool {
		_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(Bucket)})

		return err == nil
	}, startTimeout, retryEvery)
}
