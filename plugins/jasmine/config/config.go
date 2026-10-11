package config

// CronSchedule defines the cron schedule of each job, in UTC.
type CronSchedule struct {
	CareReminder string `default:"0 9 * * *" mapstructure:"care_reminder" validate:"cron"`
}

// Notification names the notification channel of each message that the plugin
// sends. An empty channel turns the message off.
type Notification struct {
	CareReminderChannel string `mapstructure:"care_reminder_channel"`
}

// Config represents the root plugin configuration.
type Config struct {
	CronSchedule CronSchedule `mapstructure:"cron_schedule"`
	Notification Notification `mapstructure:"notification"`
}
