package mcpserver_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/mcpserver"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	probePluginID = "dev.maroid.probe"
	waterToolName = "water"
)

type plantInput struct {
	Plant string `json:"plant" jsonschema:"required"`
}

type plantOutput struct {
	Watered string `json:"watered"`
}

func probeID(t *testing.T) *pluginapi.PluginID {
	t.Helper()

	id := pluginapi.ParsePluginID(probePluginID)
	require.NotNil(t, id)

	return id
}

// waterTool builds one tool whose handler answers with what the test gives.
func waterTool(
	handle func(context.Context, plantInput) (plantOutput, error),
) *pluginapi.TypedTool[plantInput, plantOutput] {
	return pluginapi.NewTypedTool(pluginapi.MCPToolMeta{
		Name:        waterToolName,
		Title:       "Water a plant",
		Description: "Record that a plant was watered.",
		Annotations: pluginapi.MCPToolAnnotations{IdempotentHint: true},
	}, handle)
}

// session installs the tool on a server and connects a client to it, the way
// mcpserver.NewServer installs every entry of the registry.
func session(t *testing.T, entry registry.MCPTool) *mcp.ClientSession {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	entry.Install(server)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession
}

func call(t *testing.T, entry registry.MCPTool, arguments string) *mcp.CallToolResult {
	t.Helper()

	result, err := session(t, entry).CallTool(t.Context(), &mcp.CallToolParams{
		Name:      entry.Name,
		Arguments: json.RawMessage(arguments),
	})
	require.NoError(t, err)

	return result
}

// listed returns the entry as an MCP client reads it in the tool list.
func listed(t *testing.T, entry registry.MCPTool) *mcp.Tool {
	t.Helper()

	result, err := session(t, entry).ListTools(t.Context(), &mcp.ListToolsParams{})
	require.NoError(t, err)

	for _, tool := range result.Tools {
		if tool.Name == entry.Name {
			return tool
		}
	}

	t.Fatalf("the tool %q is not in the list", entry.Name)

	return nil
}

// MCPHUB-SC-009: The name of a tool carries the plugin identifier, so two plugins
// that each declare "water" never collide. MCPHUB-INV-002 holds by construction.
func TestThePluginIdentifierNamesTheTool(t *testing.T) {
	t.Parallel()

	entry, err := mcpserver.NewPluginTool(probeID(t), waterTool(
		func(_ context.Context, in plantInput) (plantOutput, error) {
			return plantOutput{Watered: in.Plant}, nil
		},
	))

	require.NoError(t, err)
	require.Equal(t, "dev_maroid_probe_water", entry.Name)

	other := pluginapi.ParsePluginID("org.other.probe")
	otherEntry, err := mcpserver.NewPluginTool(other, waterTool(
		func(_ context.Context, in plantInput) (plantOutput, error) {
			return plantOutput{Watered: in.Plant}, nil
		},
	))

	require.NoError(t, err)
	require.NotEqual(t, entry.Name, otherEntry.Name)

	toolRegistry := registry.NewMCPToolRegistry()
	require.NoError(t, toolRegistry.Register(entry, otherEntry))
	require.Len(t, toolRegistry.All(), 2)
}

// MCPHUB-SC-008: A tool of a plugin answers a call with the shape that the plugin
// declared, as the structured content of the result.
func TestAToolOfAPluginAnswersItsOwnOutput(t *testing.T) {
	t.Parallel()

	entry, err := mcpserver.NewPluginTool(probeID(t), waterTool(
		func(_ context.Context, in plantInput) (plantOutput, error) {
			return plantOutput{Watered: in.Plant}, nil
		},
	))
	require.NoError(t, err)

	result := call(t, entry, `{"plant":"basil"}`)

	require.False(t, result.IsError)

	encoded, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	require.JSONEq(t, `{"watered":"basil"}`, string(encoded))
}

// MCPHUB-SC-008: The hub validates the arguments against the schema, so a call
// that does not match never reaches the tool of the plugin.
func TestAToolOfAPluginNeverSeesArgumentsThatTheSchemaRefuses(t *testing.T) {
	t.Parallel()

	entry, err := mcpserver.NewPluginTool(probeID(t), waterTool(
		func(_ context.Context, _ plantInput) (plantOutput, error) {
			t.Error("the tool must not run")

			return plantOutput{}, nil
		},
	))
	require.NoError(t, err)

	result := call(t, entry, `{"plant":42}`)

	require.True(t, result.IsError)
}

// MCPHUB-SC-014: A plugin whose settings the acting user did not fill answers
// with a failure that names the plugin and the place to fill them.
func TestAnAbsentSettingNamesThePluginAndItsSettingsPage(t *testing.T) {
	t.Parallel()

	entry, err := mcpserver.NewPluginTool(probeID(t), waterTool(
		func(_ context.Context, _ plantInput) (plantOutput, error) {
			return plantOutput{}, pluginapi.ErrSettingsAbsent
		},
	))
	require.NoError(t, err)

	result := call(t, entry, `{"plant":"basil"}`)

	require.True(t, result.IsError)
	require.Len(t, result.Content, 1)

	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	require.Contains(t, text.Text, probePluginID)
	require.Contains(t, text.Text, "/plugins/"+probePluginID+"/settings")
}

// MCPHUB-SC-013: A tool that only reports carries the read only annotation, and a
// tool that changes data does not. MCPHUB-DD-014: The plugin declares the mark.
func TestTheReadOnlyAnnotationReportsWhatThePluginDeclared(t *testing.T) {
	t.Parallel()

	open := true

	cases := map[string]pluginapi.MCPToolAnnotations{
		"a tool that writes":  {IdempotentHint: true, OpenWorldHint: &open},
		"a tool that reports": {ReadOnlyHint: true},
	}

	for name, declared := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tool := pluginapi.NewTypedTool(
				pluginapi.MCPToolMeta{Name: waterToolName, Annotations: declared},
				func(_ context.Context, _ plantInput) (plantOutput, error) {
					return plantOutput{}, nil
				},
			)

			entry, err := mcpserver.NewPluginTool(probeID(t), tool)
			require.NoError(t, err)

			found := listed(t, entry)

			require.NotNil(t, found.Annotations)
			require.Equal(t, declared.ReadOnlyHint, found.Annotations.ReadOnlyHint)
			require.Equal(t, declared.IdempotentHint, found.Annotations.IdempotentHint)
			require.Equal(t, declared.OpenWorldHint, found.Annotations.OpenWorldHint)
		})
	}
}

// MCPHUB-FR-012: A plugin that declares no hint gets the answer that makes an
// agent ask first: the tool writes, and it may destroy.
func TestAToolThatDeclaresNoHintReadsAsOneThatWrites(t *testing.T) {
	t.Parallel()

	entry, err := mcpserver.NewPluginTool(probeID(t), pluginapi.NewTypedTool(
		pluginapi.MCPToolMeta{Name: waterToolName},
		func(_ context.Context, _ plantInput) (plantOutput, error) {
			return plantOutput{}, nil
		},
	))
	require.NoError(t, err)

	found := listed(t, entry)

	require.False(t, found.Annotations.ReadOnlyHint)
	require.Nil(t, found.Annotations.DestructiveHint, "the protocol default of true stands")
}

// MCPHUB-SC-010: The report names the plugin that declares a tool.
func TestTheDescriptionNamesThePlugin(t *testing.T) {
	t.Parallel()

	entry, err := mcpserver.NewPluginTool(probeID(t), waterTool(
		func(_ context.Context, _ plantInput) (plantOutput, error) {
			return plantOutput{}, nil
		},
	))
	require.NoError(t, err)

	require.Contains(t, listed(t, entry).Description, probePluginID)
}
