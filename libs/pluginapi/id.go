package pluginapi

import (
	"fmt"
	"regexp"
	"strings"
)

var pluginIDRegex = regexp.MustCompile(
	`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)*$`,
)

// PluginID represents a parsed plugin identifier.
type PluginID struct {
	Namespace string
	Name      string
}

// ParsePluginID splits a plugin ID (e.g. "dev.maroid.foo")
// into a PluginID struct with Namespace ("dev.maroid")
// and Name ("foo").
func ParsePluginID(rawID string) *PluginID {
	if !pluginIDRegex.MatchString(rawID) {
		return nil
	}

	lastDot := strings.LastIndex(rawID, ".")
	if lastDot == -1 {
		return &PluginID{Name: rawID}
	}

	return &PluginID{
		Namespace: rawID[:lastDot],
		Name:      rawID[lastDot+1:],
	}
}

func (i *PluginID) String() string {
	if i.Namespace == "" {
		return i.Name
	}

	return fmt.Sprintf("%s.%s", i.Namespace, i.Name)
}

// ToSafeName converts the plugin ID to a safe string by replacing dots and hyphens
// with the specified separator. Useful for creating filesystem-safe or URL-safe names.
//
// Example:
//
//	id.ToSafeName("_") // "dev_maroid_foo"
func (i *PluginID) ToSafeName(separator string) string {
	replacer := strings.NewReplacer(".", separator, "-", separator)

	return replacer.Replace(i.String())
}
