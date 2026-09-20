package registrar_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/plugin/registrar"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	toolPluginID  = "dev.maroid.probe"
	pluginVersion = "0.1.0"
	getCodeName   = "get_code"
)

type codeInput struct{}

type codeOutput struct {
	Code string `json:"code"`
}

// toolPlugin is a plugin that declares the tools the test gives it.
type toolPlugin struct {
	names []string
	err   error
}

var (
	_ pluginapi.Plugin        = (*toolPlugin)(nil)
	_ pluginapi.MCPToolPlugin = (*toolPlugin)(nil)
)

func (p *toolPlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID(toolPluginID),
		Version:    pluginVersion,
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *toolPlugin) MCPTools() ([]pluginapi.MCPTool, error) {
	if p.err != nil {
		return nil, p.err
	}

	tools := make([]pluginapi.MCPTool, 0, len(p.names))
	for _, name := range p.names {
		tools = append(tools, pluginapi.NewTypedTool(
			pluginapi.MCPToolMeta{Name: name},
			func(_ context.Context, _ codeInput) (codeOutput, error) {
				return codeOutput{Code: "1234"}, nil
			},
		))
	}

	return tools, nil
}

// plainPlugin declares no tool.
type plainPlugin struct{}

var _ pluginapi.Plugin = (*plainPlugin)(nil)

func (p *plainPlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID("dev.maroid.plain"),
		Version:    pluginVersion,
		APIVersion: pluginapi.APIVersion,
	}
}

// MCPHUB-SC-008: A loaded plugin declares two tools, and both reach the registry
// that mcpserver.NewServer installs.
func TestTheRegistrarPutsEveryToolOfAPluginIntoTheRegistry(t *testing.T) {
	t.Parallel()

	toolRegistry := registry.NewMCPToolRegistry()
	reg := registrar.NewMCPToolRegistrar(toolRegistry)

	plugin := &toolPlugin{names: []string{getCodeName, "get_qr"}}

	require.True(t, reg.Supports(plugin))
	require.NoError(t, reg.Register(plugin))

	names := make([]string, 0, 2)
	for _, entry := range toolRegistry.All() {
		names = append(names, entry.Name)
	}

	require.ElementsMatch(
		t,
		[]string{"dev_maroid_probe_get_code", "dev_maroid_probe_get_qr"},
		names,
	)
}

// MCPHUB-FR-007: A plugin that declares no tool loads unchanged, because the
// registrar answers false for it. The plugin API version stays v1.
func TestTheRegistrarSkipsAPluginThatDeclaresNoTool(t *testing.T) {
	t.Parallel()

	reg := registrar.NewMCPToolRegistrar(registry.NewMCPToolRegistry())

	require.False(t, reg.Supports(&plainPlugin{}))
}

// MCPHUB-SC-009: One plugin that declares two tools under one name fails the
// load, so the registry never holds a name twice. MCPHUB-INV-002.
func TestTheRegistrarRefusesTwoToolsOfOnePluginUnderOneName(t *testing.T) {
	t.Parallel()

	toolRegistry := registry.NewMCPToolRegistry()
	reg := registrar.NewMCPToolRegistrar(toolRegistry)

	err := reg.Register(&toolPlugin{names: []string{getCodeName, getCodeName}})

	require.Error(t, err)
}
