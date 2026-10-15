package pluginapi

import (
	"context"
	"errors"
)

// ErrPageAddressAbsent reports that the host cannot build the address of a page: the
// configuration names no deck, or the unit of work carries no acting workspace.
var ErrPageAddressAbsent = errors.New("page: no address")

// PluginPages builds the deck address of a page of one plugin, in the acting workspace.
type PluginPages interface {
	// PageURL answers the address of the page route path of the plugin.
	PageURL(ctx context.Context, path string) (string, error)
}
