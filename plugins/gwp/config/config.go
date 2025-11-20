// Package config defines the configuration schema for the plugin.
package config

// CronSchedule defines the cron schedule configuration for different jobs.
type CronSchedule struct {
	ReadingsCollector string `default:"0 0 15 * *" mapstructure:"readings_collector" validate:"cron"`
}

// Config represents the root plugin configuration.
type Config struct {
	BaseURL      string       `default:"https://www.gwp.ge/api" mapstructure:"base_url"`
	Username     string       `                                 mapstructure:"username"      validate:"required"`
	Password     string       `                                 mapstructure:"password"      validate:"required"`
	CronSchedule CronSchedule `                                 mapstructure:"cron_schedule"`
}
