package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// Wrapper wraps a TelegramCommand to modify its Meta information with plugin ID.
type Wrapper struct {
	cmd      pluginapi.TelegramCommand
	pluginID *pluginapi.PluginID
}

var _ pluginapi.TelegramCommand = (*Wrapper)(nil)

// NewWrapper creates a wrapper for given command.
func NewWrapper(cmd pluginapi.TelegramCommand, pluginID *pluginapi.PluginID) *Wrapper {
	return &Wrapper{
		cmd:      cmd,
		pluginID: pluginID,
	}
}

// Meta modifies the underlying command's Meta to include plugin ID.
func (w *Wrapper) Meta() pluginapi.TelegramCommandMeta {
	meta := w.cmd.Meta()
	meta.Command = fmt.Sprintf("%s_%s", w.pluginID.ToSafeName("_"), meta.Command)
	meta.Description = fmt.Sprintf("%s (plugin %s)", meta.Description, w.pluginID.String())

	return meta
}

// Validate executes the underlying command's Validate method.
func (w *Wrapper) Validate(update telego.Update) error {
	return w.cmd.Validate(update) //nolint:wrapcheck
}

// Handle executes the underlying command's Handle method.
func (w *Wrapper) Handle(ctx *th.Context, update telego.Update) error {
	return w.cmd.Handle(ctx, update) //nolint:wrapcheck
}
