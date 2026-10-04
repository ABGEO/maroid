package tools

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/mcpserver"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
)

const savePluginSettingsName = "save_plugin_settings"

// SavePluginSettingsInput carries the values to store.
type SavePluginSettingsInput struct {
	Workspace string         `json:"workspace" jsonschema:"the workspace of the call, from list_workspaces"`
	Plugin    string         `json:"plugin"    jsonschema:"the plugin identifier"`
	Values    map[string]any `json:"values"    jsonschema:"one member for each field to store"`
}

// SavePluginSettingsOutput reports the settings that the row holds after the save.
//
//nolint:tagliatelle
type SavePluginSettingsOutput struct {
	Plugin       string         `json:"plugin"       jsonschema:"the plugin identifier"`
	SecretFields []string       `json:"secretFields" jsonschema:"the key of each secret field"`
	Values       map[string]any `json:"values"       jsonschema:"the value of each stored field"`
}

// savePluginSettings holds what the save reads and writes.
type savePluginSettings struct {
	settingsAccess
}

// NewSavePluginSettings builds the tool that stores the settings of a plugin.
func NewSavePluginSettings(
	logger *slog.Logger,
	settingsSvc settings.Service,
	enablements workspace.EnablementChecker,
) registry.MCPTool {
	tool := &savePluginSettings{settingsAccess{
		logger: logger, settingsSvc: settingsSvc, enablements: enablements,
	}}

	return registry.MCPTool{
		Name:            savePluginSettingsName,
		ActsInWorkspace: true,
		Permission:      authz.PermissionSettingsWrite,
		Install: func(server *mcp.Server) {
			mcp.AddTool(server, &mcp.Tool{
				Name:  savePluginSettingsName,
				Title: "Store the settings of a plugin",
				Description: "Store the settings of one plugin in a workspace. A field " +
					"that this call does not name keeps its stored value, and a null " +
					"removes it. A secret field changes in the deck only, and a call that " +
					"changes one stores nothing.",
				Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
			}, tool.handle)
		},
	}
}

func (t *savePluginSettings) handle(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input SavePluginSettingsInput,
) (*mcp.CallToolResult, SavePluginSettingsOutput, error) {
	if err := t.enabled(ctx, input.Plugin); err != nil {
		return nil, SavePluginSettingsOutput{}, err
	}

	if err := t.refuseSecrets(ctx, input); err != nil {
		return nil, SavePluginSettingsOutput{}, err
	}

	if _, err := t.settingsSvc.Save(ctx, input.Plugin, input.Values, nil); err != nil {
		return nil, SavePluginSettingsOutput{}, t.failure(ctx, input.Plugin, err)
	}

	secretFields, values, err := t.stored(ctx, input.Plugin)
	if err != nil {
		return nil, SavePluginSettingsOutput{}, err
	}

	return nil, SavePluginSettingsOutput{
		Plugin:       input.Plugin,
		SecretFields: secretFields,
		Values:       values,
	}, nil
}

func (t *savePluginSettings) refuseSecrets(
	ctx context.Context,
	input SavePluginSettingsInput,
) error {
	changed, err := t.settingsSvc.ChangedSecrets(input.Plugin, input.Values)
	if err != nil {
		return t.failure(ctx, input.Plugin, err)
	}

	if len(changed) == 0 {
		return nil
	}

	return fmt.Errorf(
		"%w: %s. Fill it at %s",
		errSecretRefused,
		strings.Join(changed, ", "),
		mcpserver.SettingsPath(input.Workspace, input.Plugin),
	)
}
