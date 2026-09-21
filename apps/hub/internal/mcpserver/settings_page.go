package mcpserver

import "fmt"

// settingsPathFormat builds the page from the plugin identifier.
const settingsPathFormat = "/plugins/%s/settings"

// SettingsPath returns the page of the deck where a person fills the settings of
// a plugin.
func SettingsPath(pluginID string) string {
	return fmt.Sprintf(settingsPathFormat, pluginID)
}
