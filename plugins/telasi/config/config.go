// Package config defines the configuration schema for the plugin.
package config

// Notification defines the notification settings.
type Notification struct {
	MonthlyBill bool `default:"true" mapstructure:"monthly_bill"`
}

// CronSchedule defines the cron schedule configuration for different jobs.
type CronSchedule struct {
	BillingItemsCollector string `default:"0 10 1 * *" mapstructure:"billing_items_collector" validate:"cron"`
}

// Config represents the root plugin configuration.
type Config struct {
	BaseURL       string       `default:"https://app.telasi.ge/api" mapstructure:"base_url"`
	Email         string       `                                    mapstructure:"email"          validate:"required"`
	Password      string       `                                    mapstructure:"password"       validate:"required"`
	AccountNumber string       `                                    mapstructure:"account_number" validate:"required"`
	CronSchedule  CronSchedule `                                    mapstructure:"cron_schedule"`
	Notification  Notification
}
