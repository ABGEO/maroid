package registrar

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// CronRegistrar is responsible for registering plugin cron jobs.
type CronRegistrar struct {
	registry     *registry.CronRegistry
	capabilities *registry.CapabilityRegistry
}

var _ Registrar = (*CronRegistrar)(nil)

// NewCronRegistrar creates a new CronRegistrar.
func NewCronRegistrar(
	reg *registry.CronRegistry,
	capabilities *registry.CapabilityRegistry,
) *CronRegistrar {
	return &CronRegistrar{
		registry:     reg,
		capabilities: capabilities,
	}
}

// Name returns the name of the registrar.
func (r *CronRegistrar) Name() string {
	return "cron"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *CronRegistrar) Supports(plugin pluginapi.Plugin) bool {
	_, ok := plugin.(pluginapi.CronPlugin)

	return ok
}

// Register handles the registration of a plugin capability.
func (r *CronRegistrar) Register(plugin pluginapi.Plugin) error {
	id := plugin.Meta().ID

	cronPlugin, ok := plugin.(pluginapi.CronPlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support Cron capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	jobs, err := registerItems(id, "cron jobs", cronPlugin.CronJobs, r.registry.Register)
	if err != nil {
		return err
	}

	items := make([]registry.CronJob, 0, len(jobs))
	for _, job := range jobs {
		meta := job.Meta()
		items = append(items, registry.CronJob{ID: meta.ID, Schedule: meta.Schedule})
	}

	r.capabilities.Record(id, registry.CapCron, items)

	return nil
}
