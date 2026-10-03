package registry

import (
	"fmt"
	"maps"
	"slices"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// CronRegistry is a registry for cron jobs.
type CronRegistry struct {
	jobs    map[string]pluginapi.CronJob
	plugins map[string]string
}

// NewCronRegistry creates a new CronRegistry.
func NewCronRegistry() *CronRegistry {
	return &CronRegistry{
		jobs:    make(map[string]pluginapi.CronJob),
		plugins: make(map[string]string),
	}
}

// RegisterOf registers one or more cron jobs of the plugin.
func (r *CronRegistry) RegisterOf(pluginID string, jobs ...pluginapi.CronJob) error {
	if err := r.Register(jobs...); err != nil {
		return err
	}

	for _, job := range jobs {
		r.plugins[job.Meta().ID] = pluginID
	}

	return nil
}

// PluginOf names the plugin of the job, or the empty string for a job of the hub.
func (r *CronRegistry) PluginOf(jobID string) string {
	return r.plugins[jobID]
}

// Register registers one or more cron jobs of the hub.
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
	return slices.Collect(maps.Values(r.jobs))
}
