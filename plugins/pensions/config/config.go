// Package config defines the configuration schema for the plugin.
package config

// CronSchedule defines the cron schedule configuration for different jobs.
type CronSchedule struct {
	ContributionsCollector string `default:"0 0 1 * *" mapstructure:"contributions_collector" validate:"cron"`
}

// Config represents the root plugin configuration.
type Config struct {
	BaseURL      string       `default:"https://api7.pensions.ge/api" mapstructure:"base_url"`
	CronSchedule CronSchedule `mapstructure:"cron_schedule"`
}

// UserSettings holds the pension account of one person, which follows them into every
// workspace.
//
//nolint:lll // the tags of one field share one line.
type UserSettings struct {
	Username string `json:"username" jsonschema:"title=Username,required"                                jsonschema_extras:"x-maroid-scope=user"`
	Password string `json:"password" jsonschema:"title=Password,format=password,writeOnly=true,required" jsonschema_extras:"x-maroid-scope=user"`
}
