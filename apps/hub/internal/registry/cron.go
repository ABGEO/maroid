package registry

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// CronRegistry is a registry for cron jobs.
type CronRegistry struct {
	jobs map[string]pluginapi.CronJob
}

// NewCronRegistry creates a new CronRegistry.
func NewCronRegistry() *CronRegistry {
	return &CronRegistry{
		jobs: make(map[string]pluginapi.CronJob),
	}
}

// Register registers one or more cron jobs.
func (r *CronRegistry) Register(jobs ...pluginapi.CronJob) error {
	for _, job := range jobs {
		id := job.Meta().ID

		if _, exists := r.jobs[id]; exists {
			return fmt.Errorf("%w: %s", errs.ErrCronAlreadyRegistered, id)
		}

		r.jobs[id] = job
	}

	return nil
}

// All returns all registered cron jobs.
func (r *CronRegistry) All() []pluginapi.CronJob {
	out := make([]pluginapi.CronJob, 0, len(r.jobs))
	for _, job := range r.jobs {
		out = append(out, job)
	}

	return out
}
