package command_test

import (
	"context"
	"testing"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/telegram/command"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const workspaceH = "01998aa0-1111-7000-8000-0000000000aa"

// countingCommand counts its runs and the acting workspace of each.
type countingCommand struct {
	runs       int
	workspaces []string
}

func (c *countingCommand) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{Command: "plants", Description: "List the plants"}
}

func (c *countingCommand) Validate(telego.Update) error { return nil }

func (c *countingCommand) Handle(ctx *th.Context, _ telego.Update) error {
	c.runs++
	c.workspaces = append(c.workspaces, pluginapi.ActingWorkspaceFromContext(ctx))

	return nil
}

// recordingPrompt records each request to pick a workspace.
type recordingPrompt struct {
	texts []string
}

func (p *recordingPrompt) Prompt(_ *th.Context, _ telego.Update, text string) error {
	p.texts = append(p.texts, text)

	return nil
}

func contextOf(ctx context.Context) *th.Context {
	return (&th.Context{}).WithContext(ctx)
}

// WSPACE-SC-015: A command of a plugin with no acting workspace asks the person to
// pick one, and the command runs zero times.
func TestACommandWithNoWorkspaceAsksForAPick(t *testing.T) {
	t.Parallel()

	inner := &countingCommand{}
	prompt := &recordingPrompt{}
	wrapper := command.NewWrapper(inner, pluginapi.ParsePluginID("dev.maroid.jasmine"), prompt)

	require.NoError(t, wrapper.Handle(contextOf(t.Context()), telego.Update{}))

	assert.Zero(t, inner.runs)
	assert.Equal(t, []string{"Pick a workspace first"}, prompt.texts)
}

// WSPACE-SC-013: A command of a plugin runs in the acting workspace of the update.
func TestACommandRunsInTheActingWorkspace(t *testing.T) {
	t.Parallel()

	inner := &countingCommand{}
	prompt := &recordingPrompt{}
	wrapper := command.NewWrapper(inner, pluginapi.ParsePluginID("dev.maroid.jasmine"), prompt)
	ctx := contextOf(pluginapi.ContextWithActingWorkspace(t.Context(), workspaceH))

	require.NoError(t, wrapper.Handle(ctx, telego.Update{}))

	assert.Equal(t, []string{workspaceH}, inner.workspaces)
	assert.Empty(t, prompt.texts)
}
