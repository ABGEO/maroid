// Package loader provides functionality to dynamically load plugins.
package loader

import (
	"fmt"
	"log/slog"
	"plugin"
	"strings"
	"unicode/utf8"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/registrar"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	tgcommand "github.com/abgeo/maroid/apps/hub/internal/telegram/command"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/idempotency"
)

// The names of the symbols that each plugin exports.
const (
	ConstructorSymbol = "New"
	MetadataSymbol    = "Meta"
)

// Loader is responsible for loading and registering plugins.
type Loader struct {
	binder  *host.Binder
	plugins *registry.PluginRegistry

	registrars []registrar.Registrar
}

// symbolLookup finds an exported symbol of a plugin. *plugin.Plugin satisfies it.
type symbolLookup interface {
	Lookup(symName string) (plugin.Symbol, error)
}

// New creates a new Loader.
func New(
	binder *host.Binder,
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	idempotency idempotency.Store,
	access handler.WorkspaceAccess,
	workspacePrompter tgcommand.Prompter,
	commandRegistry *registry.CommandRegistry,
	cronRegistry *registry.CronRegistry,
	handlerRegistry *handler.Registry,
	migrationRegistry *registry.MigrationRegistry,
	mqttSubscriberRegistry *registry.MQTTSubscriberRegistry,
	pluginRegistry *registry.PluginRegistry,
	telegramCommandRegistry *registry.TelegramCommandRegistry,
	telegramConversationRegistry *registry.TelegramConversationRegistry,
	uiRegistry *registry.UIRegistry,
	settingsRegistry *registry.SettingsRegistry,
	mcpToolRegistry *registry.MCPToolRegistry,
	capabilityRegistry *registry.CapabilityRegistry,
	permissionRegistry *registry.PermissionRegistry,
	notificationTopicRegistry *registry.NotificationTopicRegistry,
	widgetRegistry *registry.WidgetRegistry,
) *Loader {
	return &Loader{
		binder:  binder,
		plugins: pluginRegistry,
		registrars: []registrar.Registrar{
			registrar.NewPluginRegistrar(pluginRegistry),
			registrar.NewPermissionRegistrar(permissionRegistry, capabilityRegistry),
			registrar.NewCommandRegistrar(commandRegistry, capabilityRegistry),
			registrar.NewCronRegistrar(cronRegistry, capabilityRegistry),
			registrar.NewHandlerRegistrar(
				logger,
				verifier,
				resolver,
				handlerRegistry,
				capabilityRegistry,
				permissionRegistry,
				idempotency,
				access,
			),
			registrar.NewMigrationRegistrar(migrationRegistry, capabilityRegistry),
			registrar.NewMQTTSubscriberRegistrar(mqttSubscriberRegistry, capabilityRegistry),
			registrar.NewTelegramCommandRegistrar(
				telegramCommandRegistry,
				capabilityRegistry,
				permissionRegistry,
				telegramChecks(logger, workspacePrompter, access),
			),
			registrar.NewTelegramConversationRegistrar(
				telegramConversationRegistry,
				capabilityRegistry,
			),
			registrar.NewUIRegistrar(uiRegistry, capabilityRegistry),
			registrar.NewSettingsRegistrar(settingsRegistry, capabilityRegistry),
			registrar.NewMCPToolRegistrar(mcpToolRegistry, capabilityRegistry, permissionRegistry),
			registrar.NewWidgetRegistrar(widgetRegistry, capabilityRegistry, permissionRegistry),
			registrar.NewNotificationTopicRegistrar(notificationTopicRegistry, capabilityRegistry),
		},
	}
}

// telegramChecks holds what a command of a plugin passes: the checks of a route of a
// plugin, and the answers of the bot.
func telegramChecks(
	logger *slog.Logger,
	prompter tgcommand.Prompter,
	access handler.WorkspaceAccess,
) tgcommand.Checks {
	return tgcommand.Checks{
		Logger:      logger,
		Prompter:    prompter,
		Replier:     tgcommand.TextReply{},
		Enablements: access.Enablements,
		Authorizer:  access.Authorizer,
	}
}

// Load opens a plugin from the given path, checks the metadata that it declares,
// builds it over a host bound to that metadata, and registers its capabilities.
func (r *Loader) Load(path string, cfg map[string]any) error {
	object, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open plugin %q: %w", path, err)
	}

	meta, constructor, err := openSymbols(path, object)
	if err != nil {
		return err
	}

	return r.load(path, meta, constructor, cfg)
}

// load runs every check before the constructor, because the constructor of a refused
// plugin must not run.
func (r *Loader) load(
	path string,
	declared *pluginapi.Metadata,
	constructor pluginapi.Constructor,
	cfg map[string]any,
) error {
	meta, err := copyMetadata(path, *declared)
	if err != nil {
		return err
	}

	if err = validateMetadata(meta); err != nil {
		return err
	}

	if r.plugins.Has(meta.ID.String()) {
		return fmt.Errorf("%w: %s (%s)", errs.ErrPluginAlreadyRegistered, meta.ID, path)
	}

	instance, err := constructor(r.binder.Bind(meta), cfg)
	if err != nil {
		return err
	}

	return r.registerCapabilities(registry.LoadedPlugin{Meta: meta, Instance: instance})
}

// copyMetadata answers the metadata with an identifier that the plugin does not share,
// so a later write by the plugin reaches no copy of the hub.
func copyMetadata(path string, meta pluginapi.Metadata) (pluginapi.Metadata, error) {
	if meta.ID == nil {
		return pluginapi.Metadata{}, fmt.Errorf("%w: plugin %q", errs.ErrInvalidPluginID, path)
	}

	parsed := pluginapi.ParsePluginID(meta.ID.String())
	if parsed == nil {
		return pluginapi.Metadata{}, fmt.Errorf(
			"%w: %q in plugin %q",
			errs.ErrInvalidPluginID,
			meta.ID.String(),
			path,
		)
	}

	meta.ID = parsed

	return meta, nil
}

// @todo: move capabilities registration to the plugin registrar.
func (r *Loader) registerCapabilities(plg registry.LoadedPlugin) error {
	for _, reg := range r.registrars {
		if !reg.Supports(plg) {
			continue
		}

		if err := reg.Register(plg); err != nil {
			return fmt.Errorf("registering capabilities for plugin %s via %s: %w",
				plg.Meta.ID, reg.Name(), err)
		}
	}

	return nil
}

func validateMetadata(meta pluginapi.Metadata) error {
	name := utf8.RuneCountInString(strings.TrimSpace(meta.Name))
	description := utf8.RuneCountInString(strings.TrimSpace(meta.Description))

	switch {
	case name == 0:
		return fmt.Errorf(
			"%w: plugin %q: the name is empty",
			errs.ErrInvalidPluginMetadata,
			meta.ID,
		)
	case name > pluginapi.MaxNameLength:
		return fmt.Errorf("%w: plugin %q: the name holds %d characters, at most %d",
			errs.ErrInvalidPluginMetadata, meta.ID, name, pluginapi.MaxNameLength)
	case description > pluginapi.MaxDescriptionLength:
		return fmt.Errorf("%w: plugin %q: the description holds %d characters, at most %d",
			errs.ErrInvalidPluginMetadata, meta.ID, description, pluginapi.MaxDescriptionLength)
	case meta.APIVersion != pluginapi.APIVersion:
		return fmt.Errorf("%w: plugin %q built for API %s, expected %s",
			errs.ErrIncompatiblePluginAPIVersion, meta.ID, meta.APIVersion, pluginapi.APIVersion)
	default:
		return nil
	}
}

func openSymbols(
	path string,
	symbols symbolLookup,
) (*pluginapi.Metadata, pluginapi.Constructor, error) {
	metaSymbol, err := symbols.Lookup(MetadataSymbol)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot find metadata in plugin %q: %w", path, err)
	}

	meta, ok := metaSymbol.(*pluginapi.Metadata)
	if !ok {
		return nil, nil, fmt.Errorf(
			"%w: %q (%s)",
			errs.ErrUnexpectedPluginSymbolType,
			MetadataSymbol,
			path,
		)
	}

	constructorSymbol, err := symbols.Lookup(ConstructorSymbol)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot find constructor in plugin %q: %w", path, err)
	}

	constructor, ok := constructorSymbol.(*pluginapi.Constructor)
	if !ok {
		return nil, nil, fmt.Errorf(
			"%w: %q (%s)",
			errs.ErrUnexpectedPluginSymbolType,
			ConstructorSymbol,
			path,
		)
	}

	return meta, *constructor, nil
}
