package registrar

import (
	"fmt"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// registerItems retrieves the items that a plugin capability provides, registers
// every one, and returns them so the caller can report the capability.
// The capability names the item kind for the error context.
func registerItems[I any](
	id *pluginapi.PluginID,
	capability string,
	items func() ([]I, error),
	register func(...I) error,
) ([]I, error) {
	list, err := items()
	if err != nil {
		return nil, fmt.Errorf("retrieving %s for plugin %s: %w", capability, id, err)
	}

	if err = register(list...); err != nil {
		return nil, fmt.Errorf("registering %s for plugin %s: %w", capability, id, err)
	}

	return list, nil
}
