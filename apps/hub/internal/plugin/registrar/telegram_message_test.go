package registrar_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/plugin/registrar"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
	messageapi "github.com/abgeo/maroid/libs/pluginapi/telegram/message"
)

// messagingProbe is a plugin that receives a press and a reply on its messages.
type messagingProbe struct{ id string }

var _ pluginapi.TelegramMessagePlugin = (*messagingProbe)(nil)

func (p *messagingProbe) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID(p.id),
		Version:    pluginVersion,
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *messagingProbe) OnTelegramPress(context.Context, messageapi.Press) (string, error) {
	return "", nil
}

func (p *messagingProbe) OnTelegramReply(context.Context, messageapi.Reply) error {
	return nil
}

// silentProbe is a plugin that receives nothing on a message.
type silentProbe struct{}

func (p *silentProbe) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID("dev.maroid.silent"),
		Version:    pluginVersion,
		APIVersion: pluginapi.APIVersion,
	}
}

// TGMSG-SC-020: The capabilities of P report telegram_messages, and a plugin with no
// receiver reports none.
func TestTelegramMessageRegistrarRecordsTheReceiver(t *testing.T) {
	t.Parallel()

	receivers := registry.NewTelegramMessageRegistry()
	capabilities := registry.NewCapabilityRegistry()
	reg := registrar.NewTelegramMessageRegistrar(receivers, capabilities)

	plugin := &messagingProbe{id: "dev.maroid.p"}
	silent := &silentProbe{}

	require.True(t, reg.Supports(loaded(plugin)))
	require.False(t, reg.Supports(loaded(silent)))
	require.NoError(t, reg.Register(loaded(plugin)))

	receiver, found := receivers.Get("dev.maroid.p")
	require.True(t, found)
	assert.Same(t, plugin, receiver)

	assert.Equal(t, registry.Present, capabilities.Of("dev.maroid.p")[registry.CapTelegramMessages])
	assert.NotContains(t, capabilities.Of("dev.maroid.silent"), registry.CapTelegramMessages)

	_, found = receivers.Get("dev.maroid.silent")
	assert.False(t, found)
}
