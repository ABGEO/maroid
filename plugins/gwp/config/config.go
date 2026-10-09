// Package config defines the configuration schema for the plugin.
package config

// CronSchedule defines the cron schedule configuration for different jobs.
type CronSchedule struct {
	ReadingsCollector string `default:"0 0 15 * *" mapstructure:"readings_collector" validate:"cron"`
}

// Config represents the root plugin configuration.
type Config struct {
	BaseURL      string       `default:"https://www.gwp.ge/api" mapstructure:"base_url"`
	CronSchedule CronSchedule `mapstructure:"cron_schedule"`
}

// WorkspaceSettings holds the GWP account that every member of a workspace shares.
//
//nolint:lll // the tags of one field share one line.
type WorkspaceSettings struct {
	Username string `json:"username" jsonschema:"title=Username,required"                                jsonschema_extras:"x-maroid-scope=workspace"`
	Password string `json:"password" jsonschema:"title=Password,format=password,writeOnly=true,required" jsonschema_extras:"x-maroid-scope=workspace"`
}
