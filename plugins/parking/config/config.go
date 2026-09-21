// Package config defines the configuration schema for the plugin.
package config

// Config represents the root plugin configuration.
type Config struct {
	BaseURL string `default:"https://api.municipal.gov.ge" mapstructure:"base_url"`
}

// UserSettings represents the fields that one user fills for the plugin.
type UserSettings struct {
	AuthToken string `json:"authToken" jsonschema:"title=Auth token,format=password,writeOnly=true,required"`
	VehicleID string `json:"vehicleId" jsonschema:"title=Vehicle identifier,required"`
}
