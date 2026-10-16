package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	// noWorkspaceText asks for the pick that a command of a plugin needs.
	noWorkspaceText = "Pick a workspace first"
	// disabledText names the plugin, the workspace that does not enable it, and the
	// role that enables it.
	disabledText = "The plugin `%s` is not enabled in the workspace `%s`. A manager can enable it"
)

// Replier answers an update with one plain text.
type Replier interface {
	Reply(ctx *th.Context, update telego.Update, text string) error
}

// ActingWorkspaceReader reads the acting workspace of the context.
type ActingWorkspaceReader interface {
	Get(ctx context.Context) (*model.Workspace, error)
}

// Checks holds what a command of a plugin passes before it runs: an acting workspace,
// the enablement of the plugin in it, and the permission of the command.
type Checks struct {
	Logger      *slog.Logger
	Prompter    Prompter
	Replier     Replier
	Workspaces  ActingWorkspaceReader
	Enablements workspace.EnablementChecker
	Authorizer  authz.Authorizer
}

// Wrapper wraps a TelegramCommand to modify its Meta information with plugin ID, and
// runs it only in an acting workspace that enables the plugin, for a role that holds
// its permission.
type Wrapper struct {
	cmd        pluginapi.TelegramCommand
	pluginID   *pluginapi.PluginID
	pluginName string
	checks     Checks
}

var _ pluginapi.TelegramCommand = (*Wrapper)(nil)

// NewWrapper creates a wrapper for the given command of the plugin that meta describes.
func NewWrapper(
	cmd pluginapi.TelegramCommand,
	meta pluginapi.Metadata,
	checks Checks,
) *Wrapper {
	return &Wrapper{cmd: cmd, pluginID: meta.ID, pluginName: meta.Name, checks: checks}
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
// not enable the plugin answers so before the permission, so the answer names no
// permission. A role that does not hold the permission of the command answers which
// role does. None of them runs the command.
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
		return w.refuseDisabled(ctx, update, workspaceID)
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

func (w *Wrapper) refuseDisabled(ctx *th.Context, update telego.Update, workspaceID string) error {
	acting, err := w.checks.Workspaces.Get(ctx)
	if err != nil {
		return fmt.Errorf("reading the acting workspace: %w", err)
	}

	w.checks.Logger.InfoContext(
		ctx,
		"refused a command of a plugin that the workspace does not enable",
		slog.String("plugin", w.pluginID.String()),
		slog.String("workspace", workspaceID),
	)

	return w.checks.Replier.Reply( //nolint:wrapcheck
		ctx,
		update,
		fmt.Sprintf(disabledText, w.pluginName, acting.Name),
	)
}
