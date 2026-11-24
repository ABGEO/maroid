// Package config defines the configuration schema for the plugin.
package config

import "time"

// CronSchedule defines the cron schedule configuration for different jobs.
type CronSchedule struct {
	TrafficCollector string `default:"0 9 * * *" mapstructure:"traffic_collector" validate:"cron"`
}

// Config represents the root plugin configuration. StateTimeout bounds one read of the
// subscriber state, so a widget answers before the hub ends it at 5 seconds.
//
//nolint:lll // the tags of one field share one line.
type Config struct {
	BaseURL        string        `default:"https://www.magticom.ge" mapstructure:"base_url"        validate:"url,endsnotwith=/"`
	RequestTimeout time.Duration `default:"15s"                     mapstructure:"request_timeout" validate:"gt=0"`
	StateTimeout   time.Duration `default:"4s"                      mapstructure:"state_timeout"   validate:"gt=0,lte=4s"`
	CronSchedule   CronSchedule  `mapstructure:"cron_schedule"`
}

// WorkspaceSettings holds the Magti account that every member of a workspace shares.
//
//nolint:lll // the tags of one field share one line.
type WorkspaceSettings struct {
	Username string `json:"username" jsonschema:"title=Phone number or username,required"                jsonschema_extras:"x-maroid-scope=workspace"`
	Password string `json:"password" jsonschema:"title=Password,format=password,writeOnly=true,required" jsonschema_extras:"x-maroid-scope=workspace"`
}
