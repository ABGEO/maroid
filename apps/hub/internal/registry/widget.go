package registry

import (
	"context"
	"fmt"
	"slices"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// Widget is one widget of one plugin. Permission holds the full name of the
// permission, and Load answers the data for the acting user.
type Widget struct {
	PluginID   string
	Meta       pluginapi.WidgetMeta
	Permission string
	Load       func(ctx context.Context) (pluginapi.WidgetAnswer, error)
}

// WidgetRegistry holds the widgets of every plugin, in the order that each plugin
// declares them.
type WidgetRegistry struct {
	entries map[string][]Widget
}

// NewWidgetRegistry creates a new WidgetRegistry.
func NewWidgetRegistry() *WidgetRegistry {
	return &WidgetRegistry{entries: make(map[string][]Widget)}
}

// Register stores the widgets, all of them or none. A key that the plugin already
// holds, or that the widgets hold twice, refuses the whole call.
func (r *WidgetRegistry) Register(widgets ...Widget) error {
	seen := make(map[[2]string]struct{}, len(widgets))

	for pluginID, held := range r.entries {
		for _, widget := range held {
			seen[[2]string{pluginID, widget.Meta.Key}] = struct{}{}
		}
	}

	for _, widget := range widgets {
		key := [2]string{widget.PluginID, widget.Meta.Key}
		if _, repeated := seen[key]; repeated {
			return fmt.Errorf(
				"%w: plugin %s, widget %s",
				errs.ErrWidgetAlreadyRegistered,
				widget.PluginID,
				widget.Meta.Key,
			)
		}

		seen[key] = struct{}{}
	}

	for _, widget := range widgets {
		r.entries[widget.PluginID] = append(r.entries[widget.PluginID], widget)
	}

	return nil
}

// Get returns one widget of one plugin.
func (r *WidgetRegistry) Get(pluginID, key string) (Widget, bool) {
	index := slices.IndexFunc(r.entries[pluginID], func(widget Widget) bool {
		return widget.Meta.Key == key
	})
	if index < 0 {
		return Widget{}, false
	}

	return r.entries[pluginID][index], true
}

// Of returns the widgets of one plugin, in the declared order.
func (r *WidgetRegistry) Of(pluginID string) []Widget {
	return slices.Clone(r.entries[pluginID])
}
