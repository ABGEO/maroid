// Package loader provides functionality to dynamically load plugins.
package loader

import (
	"fmt"
	"plugin"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/registrar"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// ConstructorSymbol is the name of the exported constructor symbol
// expected in each plugin.
const ConstructorSymbol = "New"

// Loader is responsible for loading and registering plugins.
type Loader struct {
	host pluginapi.Host

	registrars []registrar.Registrar
}

// New creates a new Loader.
func New(
	host pluginapi.Host,
	cfg *config.Config,
	jwtSvc *auth.JWTService,
	commandRegistry *registry.CommandRegistry,
	cronRegistry *registry.CronRegistry,
	handlerRegistry *handler.Registry,
	migrationRegistry *registry.MigrationRegistry,
	mqttSubscriberRegistry *registry.MQTTSubscriberRegistry,
	pluginRegistry *registry.PluginRegistry,
	telegramCommandRegistry *registry.TelegramCommandRegistry,
	telegramConversationRegistry *registry.TelegramConversationRegistry,
	uiRegistry *registry.UIRegistry,
) *Loader {
	logger := host.Logger()

	return &Loader{
		host: host,

		registrars: []registrar.Registrar{
			registrar.NewPluginRegistrar(pluginRegistry),
			registrar.NewCommandRegistrar(commandRegistry),
			registrar.NewCronRegistrar(cronRegistry),
			registrar.NewHandlerRegistrar(logger, cfg, jwtSvc, handlerRegistry),
			registrar.NewMigrationRegistrar(migrationRegistry),
			registrar.NewMQTTSubscriberRegistrar(mqttSubscriberRegistry),
			registrar.NewTelegramCommandRegistrar(telegramCommandRegistry),
			registrar.NewTelegramConversationRegistrar(telegramConversationRegistry),
			registrar.NewUIRegistrar(uiRegistry),
		},
	}
}

// Load opens, initializes, and validates a plugin from the given path.
// It uses the provided host and configuration map for plugin initialization.
// Loaded plugins are registered and their capabilities are set up.
func (r *Loader) Load(path string, cfg map[string]any) error {
	constructor, err := openConstructor(path)
	if err != nil {
		return err
	}

	plg, err := constructor(r.host, cfg)
	if err != nil {
		return err
	}

	if err = validatePlugin(plg); err != nil {
		return err
	}

	if err = r.registerCapabilities(plg); err != nil {
		return err
	}

	return nil
}

// @todo: move capabilities registration to the plugin registrar.
func (r *Loader) registerCapabilities(plg pluginapi.Plugin) error {
	for _, reg := range r.registrars {
		if !reg.Supports(plg) {
			continue
		}

		if err := reg.Register(plg); err != nil {
			return fmt.Errorf("registering capabilities for plugin %s via %s: %w",
				plg.Meta().ID, reg.Name(), err)
		}
	}

	return nil
}

func openConstructor(path string) (pluginapi.Constructor, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open plugin %q: %w", path, err)
	}

	symbol, err := p.Lookup(ConstructorSymbol)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot find constructor in plugin %q: %w",
			path,
			err,
		)
	}

	constructor, ok := symbol.(*pluginapi.Constructor)
	if !ok {
		return nil, fmt.Errorf(
			"%w: %q (%s)",
			errs.ErrUnexpectedPluginSymbolType,
			ConstructorSymbol,
			path,
		)
	}

	return *constructor, nil
}

func validatePlugin(plg pluginapi.Plugin) error {
	meta := plg.Meta()

	if meta.ID == nil {
		return errs.ErrInvalidPluginID
	}

	if meta.APIVersion != pluginapi.APIVersion {
		return fmt.Errorf("%w: plugin %q built for API %s, expected %s",
			errs.ErrIncompatiblePluginAPIVersion, meta.ID, meta.APIVersion, pluginapi.APIVersion)
	}

	return nil
}
