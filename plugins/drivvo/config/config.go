// Package config defines the configuration schema for the plugin.
package config

import "time"

// CronSchedule defines the cron schedule configuration for different jobs.
type CronSchedule struct {
	CarLogCollector string `default:"0 * * * *" mapstructure:"car_log_collector" validate:"cron"`
}

// Config represents the root plugin configuration. AppVersion and AppVersionName are the
// version of the web site that the requests name, because the car log service checks them.
//
//nolint:lll // the tags of one field share one line.
type Config struct {
	BaseURL        string        `default:"https://api.drivvo.com" mapstructure:"base_url"         validate:"url,endsnotwith=/"`
	RequestTimeout time.Duration `default:"15s"                    mapstructure:"request_timeout"  validate:"gt=0"`
	AppVersion     string        `default:"23"                     mapstructure:"app_version"      validate:"numeric"`
	AppVersionName string        `default:"14.2.3"                 mapstructure:"app_version_name" validate:"required"`
	CronSchedule   CronSchedule  `mapstructure:"cron_schedule"`
}

// WorkspaceSettings holds the car log account that every member of a workspace shares.
//
//nolint:lll // the tags of one field share one line.
type WorkspaceSettings struct {
	Email    string `json:"email"    jsonschema:"title=Email address,format=email,required"              jsonschema_extras:"x-maroid-scope=workspace"`
	Password string `json:"password" jsonschema:"title=Password,format=password,writeOnly=true,required" jsonschema_extras:"x-maroid-scope=workspace"`
}
