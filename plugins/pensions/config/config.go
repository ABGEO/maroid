// Package config defines the configuration schema for the plugin.
package config

import "time"

// CronSchedule defines the cron schedule configuration for different jobs.
type CronSchedule struct {
	AccountCollector string `default:"0 10 1 * *" mapstructure:"account_collector" validate:"cron"`
}

// Config represents the root plugin configuration.
//
//nolint:lll // the tags of one field share one line.
type Config struct {
	BaseURL        string        `default:"https://api7.pensions.ge/api" mapstructure:"base_url"        validate:"url,endsnotwith=/"`
	RequestTimeout time.Duration `default:"15s"                          mapstructure:"request_timeout" validate:"gt=0"`
	CronSchedule   CronSchedule  `mapstructure:"cron_schedule"`
}

// UserSettings holds the pension account of one person, which follows them into every
// workspace.
//
//nolint:lll // the tags of one field share one line.
type UserSettings struct {
	Username string `json:"username" jsonschema:"title=Username,required"                                jsonschema_extras:"x-maroid-scope=user"`
	Password string `json:"password" jsonschema:"title=Password,format=password,writeOnly=true,required" jsonschema_extras:"x-maroid-scope=user"`
}
