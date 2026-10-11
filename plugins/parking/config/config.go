// Package config defines the configuration schema for the plugin.
package config

// Config represents the root plugin configuration.
type Config struct {
	BaseURL string `default:"https://api.municipal.gov.ge" mapstructure:"base_url"`
}

// UserSettings represents the fields that one user fills for the plugin: the
// credentials of their parking account.
//
//nolint:lll // the tags of one field share one line.
type UserSettings struct {
	Username string `json:"username" jsonschema:"title=Username,required"                                jsonschema_extras:"x-maroid-scope=user"`
	Password string `json:"password" jsonschema:"title=Password,format=password,writeOnly=true,required" jsonschema_extras:"x-maroid-scope=user"`
}
