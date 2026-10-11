package pluginapi

import (
	"context"
	"errors"
)

// ErrPageAddressAbsent reports that the host cannot build the address of a page: the
// configuration names no deck, or the unit of work carries no acting workspace.
var ErrPageAddressAbsent = errors.New("page: no address")

// PageLinker builds the deck address of a page of a plugin, in the acting workspace.
// The hub implements it, and Host hands it to a plugin.
type PageLinker interface {
	// PageURL answers the address of the page route path of the plugin, for example
	// "/account".
	PageURL(ctx context.Context, pluginID *PluginID, path string) (string, error)
}
