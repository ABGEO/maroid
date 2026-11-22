package registrar

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// TelegramMessageRegistrar registers the plugin that receives a press and a reply on
// its plugin messages.
type TelegramMessageRegistrar struct {
	registry     *registry.TelegramMessageRegistry
	capabilities *registry.CapabilityRegistry
}

var _ Registrar = (*TelegramMessageRegistrar)(nil)

// NewTelegramMessageRegistrar creates a new TelegramMessageRegistrar.
func NewTelegramMessageRegistrar(
	reg *registry.TelegramMessageRegistry,
	capabilities *registry.CapabilityRegistry,
) *TelegramMessageRegistrar {
	return &TelegramMessageRegistrar{
		registry:     reg,
		capabilities: capabilities,
	}
}

// Name returns the name of the registrar.
func (r *TelegramMessageRegistrar) Name() string {
	return "telegram_message"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *TelegramMessageRegistrar) Supports(plugin registry.LoadedPlugin) bool {
	_, ok := plugin.Instance.(pluginapi.TelegramMessagePlugin)

	return ok
}

// Register handles the registration of a plugin capability.
func (r *TelegramMessageRegistrar) Register(plugin registry.LoadedPlugin) error {
	id := plugin.Meta.ID

	receiver, ok := plugin.Instance.(pluginapi.TelegramMessagePlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support Telegram Message capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	r.registry.Register(id.String(), receiver)
	r.capabilities.Record(id, registry.CapTelegramMessages, registry.Present)

	return nil
}
