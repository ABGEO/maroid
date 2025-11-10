package pluginapi

import "context"

// CronJobMeta represents the CronJob metadata.
type CronJobMeta struct {
	ID       string
	Schedule string
}

// CronPlugin is a plugin that can register scheduled cron jobs.
type CronPlugin interface {
	Plugin
	CronJobs() ([]CronJob, error)
}

// CronJob represents a scheduled task that can be run by a cron scheduler.
type CronJob interface {
	Meta() CronJobMeta
	Run(ctx context.Context) error
}
