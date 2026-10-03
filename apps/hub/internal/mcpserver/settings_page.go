package mcpserver

import "fmt"

// settingsPathFormat builds the page from the workspace and the plugin identifier.
const settingsPathFormat = "/w/%s/plugins/%s/settings"

// SettingsPath returns the page of the deck where a member fills the settings of a
// plugin in a workspace.
func SettingsPath(workspaceID string, pluginID string) string {
	return fmt.Sprintf(settingsPathFormat, workspaceID, pluginID)
}
