package command

import (
	"fmt"
	"log/slog"

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

// Checks holds what a command of a plugin passes before it runs: an acting workspace,
// the enablement of the plugin in it, and the permission of the command.
type Checks struct {
	Logger      *slog.Logger
	Prompter    Prompter
	Replier     Replier
	Enablements workspace.EnablementChecker
	Authorizer  authz.Authorizer
}

// Wrapper wraps a TelegramCommand to modify its Meta information with plugin ID, and
// runs it only in an acting workspace that enables the plugin, for a role that holds
// its permission.
type Wrapper struct {
	cmd      pluginapi.TelegramCommand
	pluginID *pluginapi.PluginID
	checks   Checks
}

var _ pluginapi.TelegramCommand = (*Wrapper)(nil)

// NewWrapper creates a wrapper for given command.
func NewWrapper(
	cmd pluginapi.TelegramCommand,
	pluginID *pluginapi.PluginID,
	checks Checks,
) *Wrapper {
	return &Wrapper{cmd: cmd, pluginID: pluginID, checks: checks}
}

// PluginID names the plugin of the command.
func (w *Wrapper) PluginID() string {
	return w.pluginID.String()
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
// An update with no acting workspace asks the person to pick one. A workspace that does
// not enable the plugin answers nothing, as a command that does not exist, so the
// check runs before the permission. A role that does not hold the permission of the
// command answers which role does. None of them runs the command.
func (w *Wrapper) Handle(ctx *th.Context, update telego.Update) error {
	workspaceID := pluginapi.ActingWorkspaceFromContext(ctx)
	if workspaceID == "" {
		return w.checks.Prompter.Prompt(ctx, update, noWorkspaceText) //nolint:wrapcheck
	}

	enabled, err := w.checks.Enablements.IsEnabled(ctx, workspaceID, w.pluginID.String())
	if err != nil {
		return fmt.Errorf("checking the enablement of the plugin: %w", err)
	}

	if !enabled {
		w.checks.Logger.InfoContext(
			ctx,
			"dropped a command of a plugin that the workspace does not enable",
			slog.String("plugin", w.pluginID.String()),
			slog.String("workspace", workspaceID),
		)

		return nil
	}

	permission := registry.PermissionName(w.pluginID, w.cmd.Meta().Permission)

	allowed, lowest, err := w.checks.Authorizer.Allowed(workspace.RoleFromContext(ctx), permission)
	if err != nil {
		return fmt.Errorf("checking the permission of the command: %w", err)
	}

	if !allowed {
		refusal := authz.Refusal(permission, lowest)

		return w.checks.Replier.Reply(ctx, update, refusal) //nolint:wrapcheck
	}

	return w.cmd.Handle(ctx, update) //nolint:wrapcheck
}
