package pluginapi

import "io/fs"

// UIPlugin is a plugin that provides a web UI.
type UIPlugin interface {
	Plugin
	UIManifest() (*UIManifest, error)
}

// UIManifest describes a plugin's UI capabilities. Routes lists the navigation
// entries, and the remote declares every page route, also the ones with a
// parameter, in its own routes object.
type UIManifest struct {
	Routes []UIRoute `json:"routes"`
	Assets fs.FS     `json:"-"`
}

// UIRoute is one navigation entry of the plugin UI. Its path holds no parameter
// segment, because the navigation opens the page with no value, and the hub refuses
// a plugin whose entry holds one.
type UIRoute struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}
