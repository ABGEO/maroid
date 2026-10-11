package registrar_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/registrar"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// pages is a plugin whose manifest lists the navigation entries that the test
// gives it.
type pages struct {
	id     string
	routes []pluginapi.UIRoute
}

var _ pluginapi.UIPlugin = (*pages)(nil)

func (p *pages) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID(p.id),
		Version:    pluginVersion,
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *pages) UIManifest() (*pluginapi.UIManifest, error) {
	return &pluginapi.UIManifest{Routes: p.routes}, nil
}

// PLROUTE-SC-007: A navigation entry with a parameter segment stops the load of
// the plugin, and an entry with fixed segments alone registers.
func TestANavigationEntryWithAParameterFailsTheLoad(t *testing.T) {
	t.Parallel()

	manifests := registry.NewUIRegistry()
	uiRegistrar := registrar.NewUIRegistrar(manifests, registry.NewCapabilityRegistry())

	refused := &pages{id: "dev.maroid.probe", routes: []pluginapi.UIRoute{
		{Path: "/plants", Label: "Plants"},
		{Path: "/plants/:id", Label: "Plant"},
	}}

	err := uiRegistrar.Register(refused)
	require.ErrorIs(t, err, errs.ErrInvalidUIRoute)
	require.ErrorContains(t, err, "/plants/:id")

	_, held := manifests.Get("dev.maroid.probe")
	assert.False(t, held, "a refused plugin leaves no manifest")

	accepted := &pages{id: "dev.maroid.other", routes: []pluginapi.UIRoute{
		{Path: "/plants", Label: "Plants"},
	}}

	require.NoError(t, uiRegistrar.Register(accepted))

	_, held = manifests.Get("dev.maroid.other")
	assert.True(t, held)
}
