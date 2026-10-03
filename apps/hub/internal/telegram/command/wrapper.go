package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// noWorkspaceText asks for the pick that a command of a plugin needs.
const noWorkspaceText = "Pick a workspace first"

// Replier answers an update with one plain text.
type Replier interface {
	Reply(ctx *th.Context, update telego.Update, text string) error
}

// Wrapper wraps a TelegramCommand to modify its Meta information with plugin ID, and
// runs it only in an acting workspace, for a role that holds its permission.
type Wrapper struct {
	cmd        pluginapi.TelegramCommand
	pluginID   *pluginapi.PluginID
	prompter   Prompter
	replier    Replier
	authorizer authz.Authorizer
}

var _ pluginapi.TelegramCommand = (*Wrapper)(nil)

// NewWrapper creates a wrapper for given command.
func NewWrapper(
	cmd pluginapi.TelegramCommand,
	pluginID *pluginapi.PluginID,
	prompter Prompter,
	replier Replier,
	authorizer authz.Authorizer,
) *Wrapper {
	return &Wrapper{
		cmd:        cmd,
		pluginID:   pluginID,
		prompter:   prompter,
		replier:    replier,
		authorizer: authorizer,
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

// Handle executes the underlying command's Handle method in the acting workspace.
// An update with no acting workspace asks the person to pick one, and a role that
// does not hold the permission of the command answers which role does. Neither runs
// the command.
func (w *Wrapper) Handle(ctx *th.Context, update telego.Update) error {
	if pluginapi.ActingWorkspaceFromContext(ctx) == "" {
		return w.prompter.Prompt(ctx, update, noWorkspaceText) //nolint:wrapcheck
	}

	permission := registry.PermissionName(w.pluginID, w.cmd.Meta().Permission)

	allowed, lowest, err := w.authorizer.Allowed(workspace.RoleFromContext(ctx), permission)
	if err != nil {
		return fmt.Errorf("checking the permission of the command: %w", err)
	}

	if !allowed {
		return w.replier.Reply(ctx, update, authz.Refusal(permission, lowest)) //nolint:wrapcheck
	}

	return w.cmd.Handle(ctx, update) //nolint:wrapcheck
}
