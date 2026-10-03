package command_test

import (
	"context"
	"testing"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/telegram/command"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	workspaceH    = "01998aa0-1111-7000-8000-0000000000aa"
	probePluginID = "dev.maroid.probe"
	notesWrite    = "notes.write"
)

// countingCommand counts its runs and the acting workspace of each.
type countingCommand struct {
	runs       int
	workspaces []string
}

func (c *countingCommand) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command: "notes", Description: "Write a note", Permission: notesWrite,
	}
}

func (c *countingCommand) Validate(telego.Update) error { return nil }

func (c *countingCommand) Handle(ctx *th.Context, _ telego.Update) error {
	c.runs++
	c.workspaces = append(c.workspaces, pluginapi.ActingWorkspaceFromContext(ctx))

	return nil
}

// recorder records each request to pick a workspace and each plain answer.
type recorder struct {
	prompts []string
	replies []string
}

func (r *recorder) Prompt(_ *th.Context, _ telego.Update, text string) error {
	r.prompts = append(r.prompts, text)

	return nil
}

func (r *recorder) Reply(_ *th.Context, _ telego.Update, text string) error {
	r.replies = append(r.replies, text)

	return nil
}

// probeAuthorizer holds the permissions of the hub and notes.write of the probe at editor.
func probeAuthorizer(t *testing.T) *authz.RoleAuthorizer {
	t.Helper()

	permissions, err := authz.NewPermissionRegistry()
	require.NoError(t, err)
	require.NoError(t, permissions.Register(registry.PermissionEntry{
		Name:   registry.PermissionName(pluginapi.ParsePluginID(probePluginID), notesWrite),
		Lowest: pluginapi.RoleEditor,
	}))

	return authz.NewRoleAuthorizer(permissions)
}

func wrapped(t *testing.T, inner *countingCommand, answers *recorder) *command.Wrapper {
	t.Helper()

	return command.NewWrapper(
		inner, pluginapi.ParsePluginID(probePluginID), answers, answers, probeAuthorizer(t),
	)
}

// contextIn is the context of an update in the workspace, with the role of the sender.
func contextIn(ctx context.Context, workspaceID string, role pluginapi.Role) *th.Context {
	if workspaceID != "" {
		ctx = workspace.ContextWithRole(
			pluginapi.ContextWithActingWorkspace(ctx, workspaceID),
			role,
		)
	}

	return (&th.Context{}).WithContext(ctx)
}

// WSPACE-SC-015: A command of a plugin with no acting workspace asks the person to
// pick one, and the command runs zero times.
func TestACommandWithNoWorkspaceAsksForAPick(t *testing.T) {
	t.Parallel()

	inner := &countingCommand{}
	answers := &recorder{}

	require.NoError(
		t,
		wrapped(t, inner, answers).Handle(contextIn(t.Context(), "", ""), telego.Update{}),
	)

	assert.Zero(t, inner.runs)
	assert.Equal(t, []string{"Pick a workspace first"}, answers.prompts)
}

// WSPACE-SC-013: A command of a plugin runs in the acting workspace of the update,
// for a role that holds its permission.
func TestACommandRunsInTheActingWorkspace(t *testing.T) {
	t.Parallel()

	inner := &countingCommand{}
	answers := &recorder{}
	ctx := contextIn(t.Context(), workspaceH, pluginapi.RoleEditor)

	require.NoError(t, wrapped(t, inner, answers).Handle(ctx, telego.Update{}))

	assert.Equal(t, []string{workspaceH}, inner.workspaces)
	assert.Empty(t, answers.prompts)
	assert.Empty(t, answers.replies)
}

// PERMS-SC-013: A command whose permission the role does not hold answers a text that
// names the permission and the role, and runs zero times.
func TestACommandOfARoleBelowItsPermissionRunsNothing(t *testing.T) {
	t.Parallel()

	inner := &countingCommand{}
	answers := &recorder{}
	ctx := contextIn(t.Context(), workspaceH, pluginapi.RoleViewer)

	require.NoError(t, wrapped(t, inner, answers).Handle(ctx, telego.Update{}))

	assert.Zero(t, inner.runs)
	require.Len(t, answers.replies, 1)
	assert.Contains(t, answers.replies[0], "dev.maroid.probe:notes.write")
	assert.Contains(t, answers.replies[0], "editor")
}
