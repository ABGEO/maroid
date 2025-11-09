package pluginapi

// Metadata contains basic information about a plugin.
type Metadata struct {
	ID         *PluginID
	Version    string
	APIVersion string
}
