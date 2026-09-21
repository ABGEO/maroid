package tools

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
)

const getPluginSettingsName = "get_plugin_settings"

// GetPluginSettingsInput names the plugin to report.
type GetPluginSettingsInput struct {
	Plugin string `json:"plugin" jsonschema:"the plugin identifier"`
}

// GetPluginSettingsOutput reports the settings schema of a plugin and the values
// that the acting user stored for it.
type GetPluginSettingsOutput struct {
	Plugin       string         `json:"plugin"       jsonschema:"the plugin identifier"`
	Schema       map[string]any `json:"schema"       jsonschema:"the JSON Schema of the fields"`
	SecretFields []string       `json:"secretFields" jsonschema:"the key of each secret field"`
	Values       map[string]any `json:"values"       jsonschema:"the value of each stored field"`
}

// getPluginSettings holds what the report reads.
type getPluginSettings struct {
	settingsAccess
}

// NewGetPluginSettings builds the tool that reports the settings of a plugin.
func NewGetPluginSettings(logger *slog.Logger, settingsSvc settings.Service) registry.MCPTool {
	tool := &getPluginSettings{settingsAccess{logger: logger, settingsSvc: settingsSvc}}

	return registry.MCPTool{
		Name: getPluginSettingsName,
		Install: func(server *mcp.Server) {
			mcp.AddTool(server, &mcp.Tool{
				Name:  getPluginSettingsName,
				Title: "Read the settings of a plugin",
				Description: "Report the settings schema of one plugin, the key of each " +
					"secret field, and the values that the acting user stored. A secret " +
					"carries a mask and never its value.",
				Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
			}, tool.handle)
		},
	}
}

func (t *getPluginSettings) handle(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetPluginSettingsInput,
) (*mcp.CallToolResult, GetPluginSettingsOutput, error) {
	document, err := t.settingsSvc.Schema(input.Plugin)
	if err != nil {
		return nil, GetPluginSettingsOutput{}, t.failure(ctx, input.Plugin, err)
	}

	var schema map[string]any

	if err = json.Unmarshal(document, &schema); err != nil {
		return nil, GetPluginSettingsOutput{}, t.failure(ctx, input.Plugin, err)
	}

	secretFields, values, err := t.stored(ctx, input.Plugin)
	if err != nil {
		return nil, GetPluginSettingsOutput{}, err
	}

	return nil, GetPluginSettingsOutput{
		Plugin:       input.Plugin,
		Schema:       schema,
		SecretFields: secretFields,
		Values:       values,
	}, nil
}
