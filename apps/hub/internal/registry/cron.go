package registry

import (
	"fmt"
	"slices"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// CronEntry is one registered cron job with the plugin that declares it. PluginID is
// empty for a job of the hub.
type CronEntry struct {
	PluginID string
	Job      pluginapi.CronJob
}

// CronRegistry is a registry for cron jobs. The identifier of a job is unique within its
// plugin, so two plugins that share no code can name a job alike.
type CronRegistry struct {
	entries []CronEntry
	keys    map[cronKey]struct{}
}

// cronKey names one job: the plugin that declares it and its identifier.
type cronKey struct {
	pluginID string
	jobID    string
}

// NewCronRegistry creates a new CronRegistry.
func NewCronRegistry() *CronRegistry {
	return &CronRegistry{keys: make(map[cronKey]struct{})}
}

// RegisterOf registers one or more cron jobs of the plugin. A job whose identifier the
// plugin already registered is refused, and nothing of the call is kept.
func (r *CronRegistry) RegisterOf(pluginID string, jobs ...pluginapi.CronJob) error {
	added := make(map[cronKey]struct{}, len(jobs))

	for _, job := range jobs {
		key := cronKey{pluginID: pluginID, jobID: job.Meta().ID}

		_, held := r.keys[key]
		_, repeated := added[key]

		if held || repeated {
			return fmt.Errorf("%w: %s", errs.ErrCronAlreadyRegistered, describe(key))
		}

		added[key] = struct{}{}
	}

	for _, job := range jobs {
		r.keys[cronKey{pluginID: pluginID, jobID: job.Meta().ID}] = struct{}{}
		r.entries = append(r.entries, CronEntry{PluginID: pluginID, Job: job})
	}

	return nil
}

// Register registers one or more cron jobs of the hub.
func (r *CronRegistry) Register(jobs ...pluginapi.CronJob) error {
	return r.RegisterOf("", jobs...)
}

// Entries returns every registered cron job with its plugin, in the order of registration.
func (r *CronRegistry) Entries() []CronEntry {
	return slices.Clone(r.entries)
}

func describe(key cronKey) string {
	if key.pluginID == "" {
		return key.jobID
	}

	return key.pluginID + "/" + key.jobID
}
