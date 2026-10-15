package registrar_test

import (
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// declaring is a fake plugin that carries the metadata that a loader would read.
type declaring interface {
	Meta() pluginapi.Metadata
}

func loaded(plugin declaring) registry.LoadedPlugin {
	return registry.LoadedPlugin{Meta: plugin.Meta(), Instance: plugin}
}
