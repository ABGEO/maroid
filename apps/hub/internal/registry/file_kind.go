package registry

import (
	"fmt"
	"slices"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// FileKind is one file kind of one plugin. Permission holds the full name of the
// permission.
type FileKind struct {
	PluginID   string
	Kind       pluginapi.FileKind
	Permission string
}

// FileKindRegistry holds the file kinds of every plugin, in the order that each
// plugin declares them.
type FileKindRegistry struct {
	entries map[string][]FileKind
}

// NewFileKindRegistry creates a new FileKindRegistry.
func NewFileKindRegistry() *FileKindRegistry {
	return &FileKindRegistry{entries: make(map[string][]FileKind)}
}

// Register stores the file kinds, all of them or none. A key that the plugin already
// holds, or that the kinds hold twice, refuses the whole call.
func (r *FileKindRegistry) Register(kinds ...FileKind) error {
	seen := make(map[[2]string]struct{}, len(kinds))

	for pluginID, held := range r.entries {
		for _, kind := range held {
			seen[[2]string{pluginID, kind.Kind.Key}] = struct{}{}
		}
	}

	for _, kind := range kinds {
		key := [2]string{kind.PluginID, kind.Kind.Key}
		if _, repeated := seen[key]; repeated {
			return fmt.Errorf(
				"%w: plugin %s, file kind %s",
				errs.ErrFileKindAlreadyRegistered,
				kind.PluginID,
				kind.Kind.Key,
			)
		}

		seen[key] = struct{}{}
	}

	for _, kind := range kinds {
		r.entries[kind.PluginID] = append(r.entries[kind.PluginID], kind)
	}

	return nil
}

// Get returns one file kind of one plugin.
func (r *FileKindRegistry) Get(pluginID, key string) (FileKind, bool) {
	index := slices.IndexFunc(r.entries[pluginID], func(kind FileKind) bool {
		return kind.Kind.Key == key
	})
	if index < 0 {
		return FileKind{}, false
	}

	return r.entries[pluginID][index], true
}

// Of returns the file kinds of one plugin, in the declared order.
func (r *FileKindRegistry) Of(pluginID string) []FileKind {
	return slices.Clone(r.entries[pluginID])
}
