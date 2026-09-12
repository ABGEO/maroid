package pluginapi

import "context"

// CronScope tells the scheduler which user a run of a job acts for.
type CronScope string

const (
	// CronScopeShared runs the job one time, with no acting user.
	// It is the value that a job carries when it declares none.
	CronScopeShared CronScope = "shared"
	// CronScopePerUser runs the job one time for each active user.
	CronScopePerUser CronScope = "per-user"
)

// CronJobMeta represents the CronJob metadata.
type CronJobMeta struct {
	ID       string
	Schedule string
	Scope    CronScope
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
