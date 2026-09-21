package mcpserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// errSettingsAbsent is the text that an agent reads when the acting user filled
// no settings for the plugin that holds the tool. The reason reaches a person
// through the agent, so it names what to do next.
var errSettingsAbsent = errors.New("the acting user holds no complete settings for the plugin")

// NewPluginTool adapts one tool of a plugin to the registry of the hub.
//
// MCPHUB-FR-008: The name carries the plugin identifier, so two plugins never
// collide. MCPHUB-FR-009: The description names the plugin that declares it.
func NewPluginTool(
	pluginID *pluginapi.PluginID,
	tool pluginapi.MCPTool,
) (registry.MCPTool, error) {
	meta := tool.Meta()

	inputSchema, err := inferSchema(meta.InputModel)
	if err != nil {
		return registry.MCPTool{}, fmt.Errorf(
			"reflecting the input of the tool %q of the plugin %s: %w", meta.Name, pluginID, err,
		)
	}

	outputSchema, err := inferSchema(meta.OutputModel)
	if err != nil {
		return registry.MCPTool{}, fmt.Errorf(
			"reflecting the output of the tool %q of the plugin %s: %w", meta.Name, pluginID, err,
		)
	}

	name := pluginToolName(pluginID, meta.Name)
	//nolint:unparam // ToolHandlerFor fixes the signature. This tool needs no CallToolResult.
	handle := func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		_ map[string]any,
	) (*mcp.CallToolResult, any, error) {
		output, handleErr := tool.Handle(ctx, req.Params.Arguments)
		if handleErr != nil {
			return nil, nil, reasonOf(pluginID, handleErr)
		}

		return nil, output, nil
	}

	return registry.MCPTool{
		Name: name,
		Install: func(server *mcp.Server) {
			mcp.AddTool(server, &mcp.Tool{
				Name:         name,
				Title:        meta.Title,
				Description:  fmt.Sprintf("%s (plugin %s)", meta.Description, pluginID),
				InputSchema:  inputSchema,
				Annotations:  annotationsOf(meta.Annotations),
				OutputSchema: outputSchema,
			}, handle)
		},
	}, nil
}

// annotationsOf carries the hints of a plugin to the shape the SDK sends. A nil
// member keeps the default that the protocol gives it.
func annotationsOf(declared pluginapi.MCPToolAnnotations) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    declared.ReadOnlyHint,
		DestructiveHint: declared.DestructiveHint,
		IdempotentHint:  declared.IdempotentHint,
		OpenWorldHint:   declared.OpenWorldHint,
	}
}

// pluginToolName builds the name that one tool of one plugin carries.
func pluginToolName(pluginID *pluginapi.PluginID, name string) string {
	return pluginID.ToSafeName("_") + "_" + name
}

// reasonOf turns the failure of a tool into the text that an agent reads.
func reasonOf(pluginID *pluginapi.PluginID, err error) error {
	if errors.Is(err, pluginapi.ErrSettingsAbsent) {
		return fmt.Errorf(
			"%w %s. Fill them at %s",
			errSettingsAbsent,
			pluginID,
			SettingsPath(pluginID.String()),
		)
	}

	return err
}
