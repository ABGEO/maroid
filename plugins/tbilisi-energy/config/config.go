// Package config defines the configuration schema for the plugin.
package config

// CronSchedule defines the cron schedule configuration for different jobs.
type CronSchedule struct {
	TransactionsCollector string `default:"0 10 1 * *" mapstructure:"transactions_collector" validate:"cron"`
}

// Config represents the root plugin configuration.
type Config struct {
	BaseURL        string       `default:"https://my.te.ge/api" mapstructure:"base_url"`
	Username       string       `                               mapstructure:"username"        validate:"required"`
	Password       string       `                               mapstructure:"password"        validate:"required"`
	CustomerNumber string       `                               mapstructure:"customer_number" validate:"required"`
	CronSchedule   CronSchedule `                               mapstructure:"cron_schedule"`
}
