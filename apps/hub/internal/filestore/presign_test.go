package filestore_test

import (
	"testing"
	"time"

	"github.com/mcuadros/go-defaults"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/filestore"
)

func offlineConfig() config.Files {
	cfg := config.Files{
		Endpoint:       "http://rustfs.storage.svc:9000",
		PublicEndpoint: "https://files.example.com",
		Bucket:         "maroid",
		AccessKey:      "maroid-hub",
		SecretKey:      "a-secret",
	}
	defaults.SetDefaults(&cfg)

	return cfg
}

// FILES-SC-038: A read address lasts at most one hour, and its expiry is at most one
// hour after the signature.
func TestAReadAddressLastsAtMostOneHour(t *testing.T) {
	t.Parallel()

	signedAt := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	client := filestore.New(
		offlineConfig(),
		filestore.WithClock(func() time.Time { return signedAt }),
	)

	address, err := client.PresignGet(t.Context(), displayKey, "")
	require.NoError(t, err)

	require.LessOrEqual(t, expiresOf(t, address.URL), 3600)
	require.Equal(t, signedAt.Add(time.Hour), address.ExpiresAt)
	require.Contains(t, address.URL, "X-Amz-Date=20261118T090000Z")
}

// FILES-SC-009: Each address names the public endpoint, so a browser reaches the file
// store and never the hub.
func TestEachAddressNamesThePublicEndpoint(t *testing.T) {
	t.Parallel()

	client := filestore.New(offlineConfig())

	upload, err := client.PresignPut(t.Context(), uploadKey, megabyte, jpegType)
	require.NoError(t, err)
	require.Contains(t, upload.URL, "https://files.example.com/maroid/"+uploadKey)
	require.Equal(t, map[string]string{"Content-Type": jpegType}, upload.Headers)

	address, err := client.PresignGet(t.Context(), displayKey, "")
	require.NoError(t, err)
	require.Contains(t, address.URL, "https://files.example.com/maroid/"+displayKey)
}
