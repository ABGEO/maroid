package registry

import "github.com/abgeo/maroid/libs/pluginapi"

// TelegramMessageRegistry holds the plugin that receives a press and a reply on the
// plugin messages of each plugin.
type TelegramMessageRegistry struct {
	receivers map[string]pluginapi.TelegramMessagePlugin
}

// NewTelegramMessageRegistry creates a new TelegramMessageRegistry.
func NewTelegramMessageRegistry() *TelegramMessageRegistry {
	return &TelegramMessageRegistry{
		receivers: make(map[string]pluginapi.TelegramMessagePlugin),
	}
}

// Register stores the receiver of the plugin.
func (r *TelegramMessageRegistry) Register(
	pluginID string,
	receiver pluginapi.TelegramMessagePlugin,
) {
	r.receivers[pluginID] = receiver
}

// Get answers the receiver of the plugin, and whether the hub loaded one.
//
//nolint:ireturn // the receiver is the plugin itself.
func (r *TelegramMessageRegistry) Get(pluginID string) (pluginapi.TelegramMessagePlugin, bool) {
	receiver, found := r.receivers[pluginID]

	return receiver, found
}
