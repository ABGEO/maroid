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
	BaseURL      string       `default:"https://app.telasi.ge/api" mapstructure:"base_url"`
	CronSchedule CronSchedule `mapstructure:"cron_schedule"`
	Notification Notification
}

// WorkspaceSettings holds the Telasi account that every member of a workspace shares.
//
//nolint:lll // the tags of one field share one line.
type WorkspaceSettings struct {
	Email         string `json:"email"          jsonschema:"title=Email,format=email,required"                      jsonschema_extras:"x-maroid-scope=workspace"`
	Password      string `json:"password"       jsonschema:"title=Password,format=password,writeOnly=true,required" jsonschema_extras:"x-maroid-scope=workspace"`
	AccountNumber string `json:"account_number" jsonschema:"title=Account number,required"                          jsonschema_extras:"x-maroid-scope=workspace"`
}
