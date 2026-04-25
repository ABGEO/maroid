package pluginapi

import "io/fs"

// UIPlugin is a plugin that provides a web UI.
type UIPlugin interface {
	Plugin
	UIManifest() (*UIManifest, error)
}

// UIManifest describes a plugin's UI capabilities.
type UIManifest struct {
	Name   string    `json:"name"`
	Routes []UIRoute `json:"routes"`
	Assets fs.FS     `json:"-"`
}

// UIRoute describes a single UI page.
type UIRoute struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}
