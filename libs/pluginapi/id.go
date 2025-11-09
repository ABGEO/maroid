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

// NewPluginIDFromString splits a plugin ID (e.g. "dev.maroid.foo")
// into a PluginID struct with Namespace ("dev.maroid")
// and Name ("foo").
func NewPluginIDFromString(rawID string) *PluginID {
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
	return fmt.Sprintf("%s.%s", i.Namespace, i.Name)
}
