// Package notice writes the answers of the bot to a failure that a person can act on.
package notice

import (
	"context"
	"errors"
	"html"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/service"
)

const (
	accountPath = "/account"

	notConnectedText = "Your parking account is not connected."
	unavailableText  = "The parking service is unavailable. Please try again later."
	noLinkText       = notConnectedText + "\n\nOpen the Account page of Tbilisi Parking in Maroid to connect it."
)

// Notice builds the answer to a failure of the parking account.
type Notice struct {
	pages    pluginapi.PageLinker
	pluginID *pluginapi.PluginID
}

// New creates a new Notice.
func New(pages pluginapi.PageLinker, pluginID *pluginapi.PluginID) *Notice {
	return &Notice{
		pages:    pages,
		pluginID: pluginID,
	}
}

// Text answers the HTML message for a failure that a person can act on: no
// connection, or a parking service that failed. It answers false for any other failure.
func (n *Notice) Text(ctx context.Context, err error) (string, bool) {
	switch {
	case errors.Is(err, service.ErrNotConnected):
		return n.notConnected(ctx), true
	case errors.Is(err, service.ErrRequestFailed):
		return unavailableText, true
	default:
		return "", false
	}
}

// notConnected links to the account page. A host that cannot build the address gets
// the name of the page, so the answer still tells the person where to go.
func (n *Notice) notConnected(ctx context.Context) string {
	address, err := n.pages.PageURL(ctx, n.pluginID, accountPath)
	if err != nil {
		return noLinkText
	}

	return notConnectedText + "\n\n" +
		`<a href="` + html.EscapeString(address) + `">Connect it in Maroid</a>.`
}
