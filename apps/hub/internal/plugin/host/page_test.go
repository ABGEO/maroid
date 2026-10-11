package host_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	deckURL     = "https://deck.example"
	workspaceID = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e78"
)

func pagesOf(t *testing.T, deck string) pluginapi.PageLinker {
	t.Helper()

	instance, err := host.New(host.Dependencies{
		Config: &config.Config{Auth: config.Auth{DeckURL: deck}},
	})
	require.NoError(t, err)

	pages, err := instance.Pages()
	require.NoError(t, err)

	return pages
}

// PARKACC-SC-020: The address joins the deck, the acting workspace, the plugin, and
// the path.
func TestPageURLJoinsTheDeckTheWorkspaceAndThePath(t *testing.T) {
	t.Parallel()

	ctx := pluginapi.ContextWithActingWorkspace(context.Background(), workspaceID)

	address, err := pagesOf(t, deckURL).
		PageURL(ctx, pluginapi.ParsePluginID("dev.maroid.parking"), "/account")

	require.NoError(t, err)
	require.Equal(t, deckURL+"/w/"+workspaceID+"/plugins/dev.maroid.parking/account", address)
}

// PARKACC-SC-021: No deck, or no acting workspace, gives no address.
func TestPageURLNeedsADeckAndAWorkspace(t *testing.T) {
	t.Parallel()

	parking := pluginapi.ParsePluginID("dev.maroid.parking")
	inWorkspace := pluginapi.ContextWithActingWorkspace(context.Background(), workspaceID)

	_, err := pagesOf(t, "").PageURL(inWorkspace, parking, "/account")
	require.ErrorIs(t, err, pluginapi.ErrPageAddressAbsent)

	_, err = pagesOf(t, deckURL).PageURL(context.Background(), parking, "/account")
	require.ErrorIs(t, err, pluginapi.ErrPageAddressAbsent)
}
