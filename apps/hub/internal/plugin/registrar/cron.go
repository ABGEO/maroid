package registrar

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// CronRegistrar is responsible for registering plugin cron jobs.
type CronRegistrar struct {
	registry *registry.CronRegistry
}

var _ Registrar = (*CronRegistrar)(nil)

// NewCronRegistrar creates a new CronRegistrar.
func NewCronRegistrar(reg *registry.CronRegistry) *CronRegistrar {
	return &CronRegistrar{
		registry: reg,
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

	jobs, err := cronPlugin.CronJobs()
	if err != nil {
		return fmt.Errorf("retrieving cron jobs for plugin %s: %w", id, err)
	}

	err = r.registry.Register(jobs...)
	if err != nil {
		return fmt.Errorf("registering cron jobs for plugin %s: %w", id, err)
	}

	return nil
}
