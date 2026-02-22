// Package config defines the configuration schema for the plugin.
package config

// Config represents the root plugin configuration.
type Config struct {
	BaseURL   string `default:"https://api.municipal.gov.ge" mapstructure:"base_url"`
	AuthToken string `                                       mapstructure:"auth_token" validate:"required"`
	VehicleID int    `                                       mapstructure:"vehicle_id" validate:"required"`
}
