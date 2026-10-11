package config

// CronSchedule defines the cron schedule of each job, in UTC.
type CronSchedule struct {
	CareReminder string `default:"0 9 * * *" mapstructure:"care_reminder" validate:"cron"`
}

// Config represents the root plugin configuration.
type Config struct {
	CronSchedule CronSchedule `mapstructure:"cron_schedule"`
}
