// Package loader provides functionality to dynamically load plugins.
package loader

import (
	"fmt"
	"plugin"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
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

	plugins    map[string]pluginapi.Plugin
	registrars []registrar.Registrar
}

// New creates a new Loader.
func New(
	host pluginapi.Host,
	commandRegistry *registry.CommandRegistry,
	cronRegistry *registry.CronRegistry,
	migrationRegistry *registry.MigrationRegistry,
	telegramCommandRegistry *registry.TelegramCommandRegistry,
	telegramConversationRegistry *registry.TelegramConversationRegistry,
	mqttSubscriberRegistry *registry.MQTTSubscriberRegistry,
) *Loader {
	return &Loader{
		host:    host,
		plugins: make(map[string]pluginapi.Plugin),
		registrars: []registrar.Registrar{
			registrar.NewCommandRegistrar(commandRegistry),
			registrar.NewCronRegistrar(cronRegistry),
			registrar.NewMigrationRegistrar(migrationRegistry),
			registrar.NewTelegramCommandRegistrar(telegramCommandRegistry),
			registrar.NewTelegramConversationRegistrar(telegramConversationRegistry),
			registrar.NewMQTTSubscriberRegistrar(mqttSubscriberRegistry),
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

	id := plg.Meta().ID.String()

	if _, exists := r.plugins[id]; exists {
		return fmt.Errorf("%w: %s", errs.ErrPluginAlreadyRegistered, id)
	}

	if err = r.registerCapabilities(plg); err != nil {
		return err
	}

	r.plugins[id] = plg

	return nil
}

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
