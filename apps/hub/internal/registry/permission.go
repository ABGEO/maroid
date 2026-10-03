package registry

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// PermissionEntry is one permission that the hub checks, under its full name.
type PermissionEntry struct {
	Name        string
	Description string
	Lowest      pluginapi.Role
}

// PermissionRegistry holds every permission of the hub and of the plugins.
type PermissionRegistry struct {
	entries map[string]PermissionEntry
}

// NewPermissionRegistry creates a new PermissionRegistry.
func NewPermissionRegistry() *PermissionRegistry {
	return &PermissionRegistry{entries: make(map[string]PermissionEntry)}
}

// PermissionName answers the full name of a permission of a plugin. The prefix comes
// from the plugin identifier, so two plugins never collide.
func PermissionName(pluginID *pluginapi.PluginID, local string) string {
	return pluginID.String() + ":" + local
}

// Register stores the entries, all of them or none. A name that the registry holds,
// or that the entries hold twice, refuses the whole call.
func (r *PermissionRegistry) Register(entries ...PermissionEntry) error {
	seen := make(map[string]struct{}, len(entries))

	for _, entry := range entries {
		_, held := r.entries[entry.Name]
		_, repeated := seen[entry.Name]

		if held || repeated {
			return fmt.Errorf("%w: %s", errs.ErrPermissionAlreadyRegistered, entry.Name)
		}

		seen[entry.Name] = struct{}{}
	}

	for _, entry := range entries {
		r.entries[entry.Name] = entry
	}

	return nil
}

// Get returns the permission with the full name.
func (r *PermissionRegistry) Get(name string) (PermissionEntry, bool) {
	entry, found := r.entries[name]

	return entry, found
}
