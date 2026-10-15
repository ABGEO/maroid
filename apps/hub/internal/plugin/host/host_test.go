package host_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// PLGBIND-SC-007: Each bind answers a host of its own for its plugin.
func TestEachBindAnswersAHostOfItsOwn(t *testing.T) {
	t.Parallel()

	binder, err := host.New(host.Dependencies{
		Config: &config.Config{Auth: config.Auth{DeckURL: deckURL}},
	})
	require.NoError(t, err)

	gwp := binder.Bind(metadataOf("dev.maroid.gwp"))
	telasi := binder.Bind(metadataOf("dev.maroid.telasi"))

	require.NotSame(t, gwp, telasi)

	ctx := pluginapi.ContextWithActingWorkspace(context.Background(), workspaceID)

	for pluginID, bound := range map[string]*host.Host{
		"dev.maroid.gwp":    gwp,
		"dev.maroid.telasi": telasi,
	} {
		pages, pagesErr := bound.Pages()
		require.NoError(t, pagesErr)

		address, addressErr := pages.PageURL(ctx, "/")
		require.NoError(t, addressErr)
		require.Contains(t, address, "/plugins/"+pluginID+"/")
	}
}

// PLGBIND-SC-015: Each record of the logger of a bound host names the plugin, its
// version, and its API version.
func TestTheLoggerNamesThePlugin(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buffer, nil))
	bound := bind(t, host.Dependencies{Logger: logger}, "dev.maroid.gwp")

	bound.Logger().Info("collected")

	record := buffer.String()
	require.Contains(t, record, "plugin=dev.maroid.gwp")
	require.Contains(t, record, "plugin_version="+pluginVersion)
	require.Contains(t, record, "plugin_api_version="+pluginapi.APIVersion)
}

// PLGBIND-SC-009: A later write to the identifier that the caller of Bind holds does
// not reach the bound host.
func TestALaterWriteToTheIdentifierDoesNotReachTheHost(t *testing.T) {
	t.Parallel()

	meta := metadataOf("dev.maroid.gwp")

	binder, err := host.New(host.Dependencies{
		Config: &config.Config{Auth: config.Auth{DeckURL: deckURL}},
	})
	require.NoError(t, err)

	bound := binder.Bind(meta)
	meta.ID.Name = "telasi"

	pages, err := bound.Pages()
	require.NoError(t, err)

	ctx := pluginapi.ContextWithActingWorkspace(context.Background(), workspaceID)

	address, err := pages.PageURL(ctx, "/")
	require.NoError(t, err)
	require.Contains(t, address, "/plugins/dev.maroid.gwp/")
}
