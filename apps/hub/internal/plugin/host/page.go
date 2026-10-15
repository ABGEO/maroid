package host

import (
	"context"
	"fmt"
	"net/url"

	"github.com/abgeo/maroid/libs/pluginapi"
)

type boundPages struct {
	deckURL  string
	pluginID *pluginapi.PluginID
}

var _ pluginapi.PluginPages = (*boundPages)(nil)

// PageURL answers <deck>/w/<acting workspace>/plugins/<plugin>/<path>, the address
// that the deck opens for a page route of a remote.
func (l *boundPages) PageURL(ctx context.Context, path string) (string, error) {
	if l.deckURL == "" {
		return "", fmt.Errorf("%w: the configuration names no deck", pluginapi.ErrPageAddressAbsent)
	}

	workspaceID := pluginapi.ActingWorkspaceFromContext(ctx)
	if workspaceID == "" {
		return "", fmt.Errorf(
			"%w: the unit of work carries no workspace",
			pluginapi.ErrPageAddressAbsent,
		)
	}

	address, err := url.JoinPath(l.deckURL, "w", workspaceID, "plugins", l.pluginID.String(), path)
	if err != nil {
		return "", fmt.Errorf("joining the address of the page: %w", err)
	}

	return address, nil
}
