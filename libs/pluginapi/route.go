package pluginapi

import "net/http"

// RoutePlugin is a plugin that can register HTTP routes.
type RoutePlugin interface {
	Plugin
	Routes() ([]Route, error)
}

// Route defines an HTTP route provided by a plugin.
type Route struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
}
