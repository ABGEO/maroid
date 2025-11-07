package pluginapi

import "strings"

// PluginID represents a parsed plugin identifier.
type PluginID struct {
	Namespace string
	Name      string
}

// ParsePluginID splits a plugin ID (e.g. "dev.maroid.foo")
// into a PluginID struct with Namespace ("dev.maroid")
// and Name ("foo").
func ParsePluginID(id string) PluginID {
	const minParts = 2

	parts := strings.Split(id, ".")
	if len(parts) < minParts {
		return PluginID{Name: id}
	}

	return PluginID{
		Namespace: strings.Join(parts[:len(parts)-1], "."),
		Name:      parts[len(parts)-1],
	}
}
