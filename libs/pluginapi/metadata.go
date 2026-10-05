package pluginapi

// The limits of the name and the description of a plugin, in characters after the
// hub trims the white space at each end.
const (
	MaxNameLength        = 64
	MaxDescriptionLength = 280
)

// Metadata contains basic information about a plugin.
type Metadata struct {
	ID          *PluginID
	Name        string
	Description string
	Version     string
	APIVersion  string
}
