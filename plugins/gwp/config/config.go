// Package config defines the configuration schema for the plugin.
package config

import "time"

// CronSchedule defines the cron schedule configuration for different jobs.
type CronSchedule struct {
	ReadingsCollector string `default:"0 0 15 * *" mapstructure:"readings_collector" validate:"cron"`
}

// Proxy is the proxy that every call to the water service goes through, when Enabled.
// Host and Port are required only then, so a configuration with no proxy loads.
//
//nolint:lll // the tags of one field share one line.
type Proxy struct {
	Enabled  bool   `default:"false"         mapstructure:"enabled"`
	Scheme   string `default:"https"         mapstructure:"scheme"                                          validate:"oneof=http https socks5"`
	Host     string `mapstructure:"host"     validate:"required_if=Enabled true,omitempty,hostname_rfc1123"`
	Port     int    `mapstructure:"port"     validate:"required_if=Enabled true,omitempty,port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// Config represents the root plugin configuration.
type Config struct {
	BaseURL        string        `default:"https://www.gwp.ge/api" mapstructure:"base_url"`
	RequestTimeout time.Duration `default:"15s"                    mapstructure:"request_timeout"`
	CronSchedule   CronSchedule  `mapstructure:"cron_schedule"`
	Proxy          Proxy         `mapstructure:"proxy"`
}

// WorkspaceSettings holds the GWP account that every member of a workspace shares.
//
//nolint:lll // the tags of one field share one line.
type WorkspaceSettings struct {
	Username string `json:"username" jsonschema:"title=Username,required"                                jsonschema_extras:"x-maroid-scope=workspace"`
	Password string `json:"password" jsonschema:"title=Password,format=password,writeOnly=true,required" jsonschema_extras:"x-maroid-scope=workspace"`
}
